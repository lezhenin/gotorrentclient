package torrent

import (
	"bytes"
	"crypto/sha1"
	"github.com/lezhenin/gotorrentclient/pkg/bitfield"
	"github.com/sirupsen/logrus"
	"log"
	"net"
	"sync"
)

type Manager struct {
	info    *Info
	state   *State
	storage *Storage

	peerId   []byte
	infoHash []byte

	seedersMap map[string]*Seeder
	mapMutex   sync.RWMutex

	pieceDownloadProgress []uint8

	downloadedPieceBitfield *bitfield.Bitfield
	downloadedBlockBitfield *bitfield.Bitfield

	downloadingBlockBitfield *bitfield.Bitfield

	lastRequestedBlock map[string]uint64

	lastPieceLength int64
	lastBlockLength int64

	blockCount int64
	pieceCount int64

	blocksPerPiece     uint8
	blocksPerLastPiece uint8

	interestingPeerCount int

	receivedMessages chan Message
	closedSeeders    chan *Seeder
	addedSeeders     chan *Seeder

	stopSignals chan struct{}
	Done        chan struct{}

	wait sync.WaitGroup
}

func NewManager(peerId, infoHash []byte, info *Info, state *State, storage *Storage) (m *Manager) {

	m = new(Manager)

	m.state = state
	m.storage = storage

	m.infoHash = infoHash
	m.peerId = peerId
	m.info = info

	m.seedersMap = make(map[string]*Seeder)
	m.receivedMessages = make(chan Message, 32)

	m.blocksPerPiece = uint8(info.PieceLength / int64(blockLength))
	m.pieceCount = info.PieceCount

	m.lastRequestedBlock = make(map[string]uint64)

	m.lastPieceLength = info.TotalLength % info.PieceLength
	if m.lastPieceLength == 0 {
		m.lastPieceLength = info.PieceLength
	}

	m.lastBlockLength = m.lastPieceLength % int64(blockLength)
	if m.lastBlockLength == 0 {
		m.lastBlockLength = int64(blockLength)
	}

	m.blocksPerLastPiece = uint8(m.lastPieceLength / int64(blockLength))
	if m.lastPieceLength%int64(blockLength) > 0 {
		m.blocksPerLastPiece += 1
	}

	m.blockCount = (m.pieceCount - 1) * int64(m.blocksPerPiece)
	m.blockCount += int64(m.blocksPerLastPiece)

	m.downloadingBlockBitfield = bitfield.NewBitfield(uint(m.blockCount))
	m.downloadedBlockBitfield = bitfield.NewBitfield(uint(m.blockCount))

	m.downloadedPieceBitfield = bitfield.NewBitfield(uint(m.pieceCount))

	m.pieceDownloadProgress = make([]uint8, m.pieceCount)
	for index := range m.pieceDownloadProgress {
		m.pieceDownloadProgress[index] = m.blocksPerPiece
	}

	m.pieceDownloadProgress[m.pieceCount-1] = m.blocksPerLastPiece

	m.Done = make(chan struct{}, 1)
	m.stopSignals = make(chan struct{}, 1)

	m.closedSeeders = make(chan *Seeder, 4)
	m.addedSeeders = make(chan *Seeder, 4)

	return m
}

func (m *Manager) AddSeeder(conn net.Conn, accept bool) (err error) {

	seeder, err := NewSeeder(m.infoHash, m.peerId, m.receivedMessages)
	if err != nil {
		return err
	}

	seeder.PeerBitfield = bitfield.NewBitfield(uint(m.info.PieceCount))

	if accept {
		err = seeder.Accept(conn)
	} else {
		err = seeder.Dial(conn)
	}

	if err != nil {
		return err
	}

	if bytes.Compare(seeder.PeerId, m.peerId) == 0 {
		seeder.Close()
		return nil
	}

	for _, connectedSeeder := range m.getSeederSlice() {
		if bytes.Compare(seeder.PeerId, connectedSeeder.PeerId) == 0 {
			seeder.Close()
			return nil
		}
	}

	m.addedSeeders <- seeder

	return nil

}

func (m *Manager) Start() {

	managerLogger.WithFields(logrus.Fields{
		"downloaded": m.state.Downloaded(),
		"uploaded":   m.state.Uploaded(),
		"left":       m.state.Left(),
		"infoHash":   m.infoHash,
	}).Info("download started")

	managerLogger.WithFields(logrus.Fields{
		"infoHash":          m.infoHash,
		"pieceCount":        m.pieceCount,
		"blockCount":        m.blockCount,
		"blockPerPiece":     m.blocksPerPiece,
		"blockPerLastPiece": m.blocksPerLastPiece,
		"lastPieceLength":   m.lastPieceLength,
		"lastBlockLength":   m.lastBlockLength,
		"totalLength":       m.info.TotalLength,
	}).Debug("download params")

	m.wait.Add(1)

	go func() {

		defer m.wait.Done()

		for {

			select {

			case seeder := <-m.addedSeeders:
				m.handleAdding(seeder)

			case seeder := <-m.closedSeeders:
				m.handleClosing(seeder)

			case message := <-m.receivedMessages:
				m.handleMessage(&message)

			case <-m.stopSignals:
				m.handleStopSignal()
				return

			}
		}
	}()

	m.wait.Wait()

}

func (m *Manager) Stop() {

	m.stopSignals <- struct{}{}

	managerLogger.WithFields(logrus.Fields{
		"downloaded": m.state.Downloaded(),
		"uploaded":   m.state.Uploaded(),
		"left":       m.state.Left(),
		"infoHash":   m.infoHash,
	}).Info("Download stopped")

}

func (m *Manager) handleStopSignal() {

	for _, seeder := range m.getSeederSlice() {
		seeder.Close()
		m.deleteSeeder(seeder.PeerId)
	}

	m.downloadingBlockBitfield =
		bitfield.And(m.downloadingBlockBitfield, m.downloadedBlockBitfield)

}

func (m *Manager) handleAdding(seeder *Seeder) {

	m.addSeeder(seeder)

	if m.state.Downloaded() > 0 {
		seeder.outcoming <- Message{Bitfield, m.state.BitfieldBytes(), m.peerId}
		managerLogger.WithFields(logrus.Fields{
			"peerId":   seeder.PeerId,
			"data":     m.state.BitfieldBytes(),
			"infoHash": m.infoHash,
		}).Info("bitfield sent")

	}

	go func() {
		seeder.Start()
		m.closedSeeders <- seeder
	}()

}

func (m *Manager) handleClosing(seeder *Seeder) {

	seeder, ok := m.getSeeder(seeder.PeerId)
	if !ok {
		return
	}

	if seeder.AmInterested == true {
		m.interestingPeerCount -= 1
	}

	seeder.Close()

	m.deleteSeeder(seeder.PeerId)

	blockIndex, ok := m.lastRequestedBlock[string(seeder.PeerId)]

	if !ok || m.downloadedBlockBitfield.Get(uint(blockIndex)) == 1 {
		return
	}

	m.downloadingBlockBitfield.Clear(uint(blockIndex))

	pieceIndex, _ := m.convertGlobalBlockToPieceIndex(int64(blockIndex))

	for _, anotherSeeder := range m.getSeederSlice() {
		if anotherSeeder.AmInterested == false && anotherSeeder.PeerBitfield.Get(uint(pieceIndex)) == 1 {
			anotherSeeder.outcoming <- Message{Interested, nil, m.peerId}
			m.interestingPeerCount += 1
		}
	}

}

func (m *Manager) handleMessage(message *Message) {

	seeder, ok := m.getSeeder(message.PeerId)

	if !ok {
		managerLogger.Debugf("Peer with id %v is not found. Ignore message.", message.PeerId)
		return
	}

	switch message.Id {

	case Bitfield:
		m.handleBitfiedMessage(seeder, message.Payload)

	case Have:
		m.handleHaveMessage(seeder, message.Payload)

	case Choke:
		m.handleChokeMessage(seeder)

	case Unchoke:
		m.handleUnchokeMessage(seeder)

	case Interested:
		m.handleInterestedMessage(seeder)

	case NotInterested:
		m.handleNotInterestedMessage(seeder)

	case Request:
		m.handleRequestMessage(seeder, message.Payload)

	case Piece:
		m.handlePieceMessage(seeder, message.Payload)

	}
}

func (m *Manager) handleBitfiedMessage(seeder *Seeder, payload []byte) {

	seeder.PeerBitfield, _ = bitfield.NewBitfieldFromBytes(payload, uint(m.pieceCount))
	interestedPieceCount := bitfield.AndNot(seeder.PeerBitfield, m.downloadedPieceBitfield).Count(1)
	if interestedPieceCount > 0 && seeder.AmInterested == false {
		seeder.AmInterested = true
		seeder.outcoming <- Message{Interested, nil, m.peerId}
		m.interestingPeerCount += 1
	}

	managerLogger.WithFields(logrus.Fields{
		"peerId":   seeder.PeerId,
		"data":     payload,
		"count":    seeder.PeerBitfield.Count(1),
		"infoHash": m.infoHash,
	}).Info("received bitfield")

}

func (m *Manager) handleHaveMessage(seeder *Seeder, payload []byte) {

	pieceIndex, err := ParseHavePayload(payload)
	if err != nil {
		seeder.Close()
		return
	}

	seeder.PeerBitfield.Set(uint(pieceIndex))
	if m.downloadedPieceBitfield.Get(uint(pieceIndex)) == 0 && seeder.AmInterested == false {
		seeder.AmInterested = true
		seeder.outcoming <- Message{Interested, nil, m.peerId}
		m.interestingPeerCount += 1
	}
}

func (m *Manager) handleChokeMessage(seeder *Seeder) {

	seeder.PeerChoking = true

}

func (m *Manager) handleUnchokeMessage(seeder *Seeder) {

	seeder.PeerChoking = false
	if seeder.AmInterested == true {
		pieceIndex, blockIndex, _ := m.requestPiece(seeder)
		index, offset, length := m.convertPieceIndexToOffset(pieceIndex, blockIndex)
		payload := MakeRequestPayload(index, offset, length)
		seeder.outcoming <- Message{Request, payload, m.peerId}
	}
}

func (m *Manager) handleInterestedMessage(seeder *Seeder) {

	seeder.PeerInterested = true
	if true { // todo condition
		seeder.AmChoking = false
		seeder.outcoming <- Message{Unchoke, nil, m.peerId}
	}
}

func (m *Manager) handleNotInterestedMessage(seeder *Seeder) {

	seeder.PeerInterested = false
	seeder.AmChoking = true
	seeder.outcoming <- Message{Choke, nil, m.peerId}
}

func (m *Manager) handleRequestMessage(seeder *Seeder, payload []byte) {

	if seeder.AmChoking == true {
		return
	}

	index, offset, length, _ := ParseRequestPayload(payload)

	//if seeder.PeerBitfield.Get(uint(index)) == 1 {
	//	return
	//}

	data := make([]byte, length)
	_, err := m.storage.ReadAt(data, int64(index)*m.info.PieceLength+int64(offset))
	if err != nil {
		log.Println(err)
		return
	}

	seeder.outcoming <- Message{Piece, MakePiecePayload(index, offset, data), m.peerId}

	m.state.IncrementUploaded(uint64(length))

}

func (m *Manager) handlePieceMessage(seeder *Seeder, payload []byte) {

	index, offset, data, err := ParsePiecePayload(payload)
	if err != nil {
		seeder.Close()
		return
	}

	pieceIndex, blockIndex := m.convertOffsetToPieceIndex(index, offset)
	m.acceptPiece(pieceIndex, blockIndex, data)

	if seeder.PeerChoking || !seeder.AmInterested {
		return
	}

	pieceIndex, blockIndex, interested := m.requestPiece(seeder)

	if interested {
		index, offset, length := m.convertPieceIndexToOffset(pieceIndex, blockIndex)
		payload := MakeRequestPayload(index, offset, length)
		seeder.outcoming <- Message{Request, payload, m.peerId}
	} else {
		seeder.outcoming <- Message{NotInterested, nil, m.peerId}
		m.interestingPeerCount -= 1
	}
}

func (m *Manager) getSeeder(peerId []byte) (seeder *Seeder, ok bool) {
	m.mapMutex.RLock()
	defer m.mapMutex.RUnlock()
	seeder, ok = m.seedersMap[string(peerId)]
	return seeder, ok
}

func (m *Manager) addSeeder(seeder *Seeder) {
	m.mapMutex.Lock()
	defer m.mapMutex.Unlock()
	m.seedersMap[string(seeder.PeerId)] = seeder
}

func (m *Manager) deleteSeeder(peerId []byte) {
	m.mapMutex.Lock()
	defer m.mapMutex.Unlock()
	delete(m.seedersMap, string(peerId))
}

func (m *Manager) getSeederSlice() (seeders []*Seeder) {
	m.mapMutex.RLock()
	defer m.mapMutex.RUnlock()
	seeders = []*Seeder{}
	for _, seeder := range m.seedersMap {
		seeders = append(seeders, seeder)
	}
	return seeders
}

func (m *Manager) convertGlobalBlockToPieceIndex(globalBlockIndex int64) (pieceIndex, blockIndex int) {
	blockIndex = int(globalBlockIndex % int64(m.blocksPerPiece))
	pieceIndex = int(globalBlockIndex / int64(m.blocksPerPiece))
	return pieceIndex, blockIndex
}

func (m *Manager) convertPieceToGlobalBlockIndex(pieceIndex, blockIndex int) (globalBlockIndex int64) {
	globalBlockIndex = int64(pieceIndex*int(m.blocksPerPiece) + blockIndex)
	return globalBlockIndex
}

func (m *Manager) convertPieceIndexToOffset(pieceIndex, blockIndex int) (index, offset, length uint32) {
	index = uint32(pieceIndex)
	offset = uint32(blockIndex * blockLength)
	if m.convertPieceToGlobalBlockIndex(pieceIndex, blockIndex) == m.blockCount-1 {
		length = uint32(m.lastBlockLength)
	} else {
		length = uint32(blockLength)
	}
	return index, offset, length
}

func (m *Manager) convertOffsetToPieceIndex(index, offset uint32) (pieceIndex, blockIndex int) {
	pieceIndex = int(index)
	blockIndex = int(offset) / blockLength
	return pieceIndex, blockIndex
}

func (m *Manager) requestPiece(seeder *Seeder) (pieceIndex, blockIndex int, interested bool) {

	index := m.downloadingBlockBitfield.GetFirstIndex(0, 0)

	for index < m.downloadingBlockBitfield.Length() {

		pieceIndex, blockIndex = m.convertGlobalBlockToPieceIndex(int64(index))

		if seeder.PeerBitfield.Get(uint(pieceIndex)) == 1 {
			m.downloadingBlockBitfield.Set(index)
			m.lastRequestedBlock[string(seeder.PeerId)] = uint64(index)

			managerLogger.WithFields(logrus.Fields{
				"pieceIndex": pieceIndex,
				"blockIndex": blockIndex,
				"infoHash":   m.infoHash,
			}).Trace("block requested")

			return pieceIndex, blockIndex, true
		}

		index = m.downloadingBlockBitfield.GetFirstIndex(index+1, 0)
	}

	return 0, 0, false

}

func (m *Manager) acceptPiece(pieceIndex, blockIndex int, data []byte) {

	managerLogger.WithFields(logrus.Fields{
		"pieceIndex": pieceIndex,
		"blockIndex": blockIndex,
		"infoHash":   m.infoHash,
	}).Trace("Block accepted")

	pieceLength := int(m.info.PieceLength)
	offset := pieceIndex*pieceLength + blockIndex*blockLength
	if _, err := m.storage.WriteAt(data, int64(offset)); err != nil {
		panic(err)
	}

	globalBlockIndex := m.convertPieceToGlobalBlockIndex(pieceIndex, blockIndex)
	m.downloadedBlockBitfield.Set(uint(globalBlockIndex))
	m.pieceDownloadProgress[pieceIndex] -= 1

	if m.pieceDownloadProgress[pieceIndex] == 0 {

		offset := pieceIndex * pieceLength

		if int64(pieceIndex) == m.pieceCount-1 {
			pieceLength = int(m.lastPieceLength)
		}

		data = make([]byte, pieceLength)
		if _, err := m.storage.ReadAt(data, int64(offset)); err != nil {
			panic(err)
		}

		pieceHash := sha1.New()
		pieceHash.Write(data)

		downloadHashSum := pieceHash.Sum(nil)
		pieceHashSum := m.info.Pieces[(20 * pieceIndex):(20*pieceIndex + 20)]

		if bytes.Compare(downloadHashSum, pieceHashSum) != 0 {

			managerLogger.WithFields(logrus.Fields{
				"pieceIndex": pieceIndex,
				"infoHash":   m.infoHash,
			}).Trace("Hash is wrong")

			if int64(pieceIndex) == m.pieceCount-1 {
				m.pieceDownloadProgress[pieceIndex] = m.blocksPerLastPiece
			} else {
				m.pieceDownloadProgress[pieceIndex] = m.blocksPerPiece
			}

			startIndex := m.convertPieceToGlobalBlockIndex(pieceIndex, 0)
			endIndex := m.convertPieceToGlobalBlockIndex(pieceIndex+1, 0)
			for i := startIndex; i < endIndex; i++ {
				m.downloadingBlockBitfield.Clear(uint(i))
			}

			return
		}

		managerLogger.WithFields(logrus.Fields{
			"pieceIndex": pieceIndex,
			"infoHash":   m.infoHash,
		}).Trace("Piece accepted")

		m.state.IncrementDownloaded(uint64(pieceLength))
		m.state.DecrementLeft(uint64(pieceLength))

		managerLogger.Tracef("Downloaded %d/%d", m.state.downloaded, m.info.TotalLength)

		m.downloadedPieceBitfield.Set(uint(pieceIndex))
		m.state.SetBitfieldBit(uint(pieceIndex))

		for _, s := range m.getSeederSlice() {
			if s.PeerBitfield.Get(uint(pieceIndex)) == 0 {
				s.outcoming <- Message{Have, MakeHavePayload(uint32(pieceIndex)), m.peerId}
			}
		}

		if m.downloadedPieceBitfield.GetFirstIndex(0, 0) == m.downloadedPieceBitfield.Length() {
			m.Done <- struct{}{}
			managerLogger.WithFields(logrus.Fields{
				"downloaded": m.state.Downloaded(),
				"uploaded":   m.state.Uploaded(),
				"left":       m.state.Left(),
				"infoHash":   m.infoHash,
			}).Info("Download completed")
		}
	}
}
