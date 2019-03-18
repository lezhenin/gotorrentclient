package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"github.com/lezhenin/gotorrentclient/bitfield"
	"github.com/lezhenin/gotorrentclient/fileoverlay"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"time"
)

//type Stage int
//
//const (
//	Stopped   Stage = 0
//	Started   Stage = 1
//	Completed Stage = 2
//)

const blockLength int = 16 * 1024

type State struct {
	uploaded   uint64
	downloaded uint64
	left       uint64
	//Stage      Stage

	mutex sync.Mutex
}

func (s *State) GetDownloadedByteCount() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.downloaded
}

func (s *State) GetUploadedByteCount() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.uploaded
}

func (s *State) GetLeftByteCount() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.left
}

type Download struct {
	Metadata          Metadata
	State             State
	PeerId            []byte
	InfoHash          []byte
	Port              uint16
	NoPeerId          bool
	ClientIP          string
	DownloadPath      string
	Files             []*os.File
	TrackerConnection *Tracker
	overlay           *fileoverlay.FileOverlay

	seedersMap map[string]*Seeder
	mapMutex   sync.RWMutex

	messages chan Message
	errors   chan error

	stop bool

	pieceDownloadProgress []uint8

	downloadedPieceBitfield  *bitfield.Bitfield
	downloadedBlockBitfield  *bitfield.Bitfield
	downloadingBlockBitfield *bitfield.Bitfield

	lastRequestedBlock map[string]uint64

	lastPieceLength int64
	lastBlockLength int64

	blockCount int64
	pieceCount int64

	blocksPerPiece     uint8
	blocksPerLastPiece uint8

	interestingPeerCount int
}

func (d *Download) getSeeder(peerId []byte) (seeder *Seeder, ok bool) {
	d.mapMutex.RLock()
	defer d.mapMutex.RUnlock()
	seeder, ok = d.seedersMap[string(peerId)]
	return seeder, ok
}

func (d *Download) addSeeder(seeder *Seeder) {
	d.mapMutex.Lock()
	defer d.mapMutex.Unlock()
	d.seedersMap[string(seeder.PeerId)] = seeder
}

func (d *Download) deleteSeeder(peerId []byte) {
	d.mapMutex.Lock()
	defer d.mapMutex.Unlock()
	delete(d.seedersMap, string(peerId))
}

func (d *Download) getSeederSlice() (seeders []*Seeder) {
	d.mapMutex.RLock()
	defer d.mapMutex.RUnlock()
	seeders = []*Seeder{}
	for _, seeder := range d.seedersMap {
		seeders = append(seeders, seeder)
	}
	return seeders
}

func (d *Download) convertGlobalBlockToPieceIndex(globalBlockIndex int64) (pieceIndex, blockIndex int) {
	blockIndex = int(globalBlockIndex % int64(d.blocksPerPiece))
	pieceIndex = int(globalBlockIndex / int64(d.blocksPerPiece))
	return pieceIndex, blockIndex
}

func (d *Download) convertPieceToGlobalBlockIndex(pieceIndex, blockIndex int) (globalBlockIndex int64) {
	globalBlockIndex = int64(pieceIndex*int(d.blocksPerPiece) + blockIndex)
	return globalBlockIndex
}

func (d *Download) convertPieceIndexToOffset(pieceIndex, blockIndex int) (index, offset, length uint32) {

	index = uint32(pieceIndex)
	offset = uint32(blockIndex * blockLength)
	if d.convertPieceToGlobalBlockIndex(pieceIndex, blockIndex) == d.blockCount-1 {
		length = uint32(d.lastBlockLength)
	} else {
		length = uint32(blockLength)
	}
	return index, offset, length
}

func (d *Download) convertOffsetToPieceIndex(index, offset uint32) (pieceIndex, blockIndex int) {
	pieceIndex = int(index)
	blockIndex = int(offset) / blockLength
	return pieceIndex, blockIndex
}

func (d *Download) requestPiece(seeder *Seeder) (pieceIndex, blockIndex int, interested bool) {

	index := d.downloadingBlockBitfield.GetFirstIndex(0, 0)

	for index < d.downloadingBlockBitfield.Length() {

		pieceIndex, blockIndex = d.convertGlobalBlockToPieceIndex(int64(index))

		if seeder.PeerBitfield.Get(uint(pieceIndex)) == 1 {
			d.downloadingBlockBitfield.Set(index)
			d.lastRequestedBlock[string(seeder.PeerId)] = uint64(index)

			log.Printf("Request block %d of piece %d", blockIndex, pieceIndex)

			return pieceIndex, blockIndex, true
		}

		index = d.downloadingBlockBitfield.GetFirstIndex(index, 0)
	}

	return 0, 0, false

}

func (d *Download) acceptPiece(pieceIndex, blockIndex int, data []byte) {

	log.Printf("Accept block %d of piece %d", blockIndex, pieceIndex)

	pieceLength := int(d.Metadata.Info.PieceLength)
	offset := pieceIndex*pieceLength + blockIndex*blockLength
	if _, err := d.overlay.WriteAt(data, int64(offset)); err != nil {
		panic(err)
	}

	globalBlockIndex := d.convertPieceToGlobalBlockIndex(pieceIndex, blockIndex)
	d.downloadedBlockBitfield.Set(uint(globalBlockIndex))
	d.pieceDownloadProgress[pieceIndex] -= 1

	if d.pieceDownloadProgress[pieceIndex] == 0 {

		offset := pieceIndex * pieceLength

		if int64(pieceIndex) == d.pieceCount-1 {
			pieceLength = int(d.lastPieceLength)
		}

		data = make([]byte, pieceLength)
		if _, err := d.overlay.ReadAt(data, int64(offset)); err != nil {
			panic(err)
		}

		pieceHash := sha1.New()
		pieceHash.Write(data)

		downloadHashSum := pieceHash.Sum(nil)
		pieceHashSum := d.Metadata.Info.Pieces[(20 * pieceIndex):(20*pieceIndex + 20)]

		if bytes.Compare(downloadHashSum, pieceHashSum) != 0 {

			log.Printf("Hash is wrong for piece %d", pieceIndex)

			if int64(pieceIndex) == d.pieceCount-1 {
				d.pieceDownloadProgress[pieceIndex] = d.blocksPerLastPiece
			} else {
				d.pieceDownloadProgress[pieceIndex] = d.blocksPerPiece
			}

			startIndex := d.convertPieceToGlobalBlockIndex(pieceIndex, 0)
			endIndex := d.convertPieceToGlobalBlockIndex(pieceIndex+1, 0)
			for i := startIndex; i < endIndex; i++ {
				d.downloadingBlockBitfield.Clear(uint(i))
			}

			return
		}

		log.Printf("Accept piece %d", pieceIndex)

		d.State.downloaded += uint64(pieceLength)
		d.State.left -= uint64(pieceLength)

		log.Printf("Downloaded %d/%d", d.State.downloaded, d.Metadata.Info.TotalLength)

		d.downloadedPieceBitfield.Set(uint(pieceIndex))

		if d.downloadedPieceBitfield.GetFirstIndex(0, 0) == d.downloadedPieceBitfield.Length() {
			log.Printf("Download finished.")
			//d.stopDownload()
		}
	}
}

func (d *Download) expressInterest(seeder *Seeder) {

	interestedPieceCount := bitfield.AndNot(seeder.PeerBitfield, d.downloadedPieceBitfield).Count(1)

	if interestedPieceCount > 0 && seeder.AmInterested == false {
		seeder.AmInterested = true
		seeder.outcoming <- Message{Interested, nil, d.PeerId}
		d.interestingPeerCount += 1
	} else if seeder.AmInterested == true {
		seeder.AmInterested = false
		seeder.outcoming <- Message{NotInterested, nil, d.PeerId}
		d.interestingPeerCount -= 1
	}
}

func (d *Download) initDownload() {

	d.pieceDownloadProgress = make([]uint8, d.pieceCount)
	for index := range d.pieceDownloadProgress {
		d.pieceDownloadProgress[index] = d.blocksPerPiece
	}
	d.pieceDownloadProgress[d.pieceCount-1] = d.blocksPerLastPiece
}

func (d *Download) startDownload() {}

func (d *Download) stopDownload() {

	for _, seeder := range d.getSeederSlice() {
		seeder.AmInterested = false
		seeder.outcoming <- Message{NotInterested, nil, d.PeerId}
		d.interestingPeerCount -= 1
		seeder.Close()
	}

	//d.Stop()
}

func (d *Download) completeDownload() {}

func (d *Download) handleMessage(message *Message) {

	seeder, ok := d.getSeeder(message.PeerId)

	if !ok {
		log.Printf("Peer with id %v is not found. Ignore message.")
		return
	}

	switch message.Id {

	case Error:

		d.deleteSeeder(message.PeerId)
		index, ok := d.lastRequestedBlock[string(message.PeerId)]
		if ok && d.downloadedBlockBitfield.Get(uint(index)) == 0 {
			d.downloadingBlockBitfield.Clear(uint(index))
		}

	case Bitfield:

		seeder.PeerBitfield, _ = bitfield.NewBitfieldFromBytes(message.Payload, uint(d.pieceCount))
		d.expressInterest(seeder)

	case Have:

		pieceIndex, err := parseHavePayload(message.Payload)
		if err != nil {
			seeder.Close()
		}
		seeder.PeerBitfield.Set(uint(pieceIndex))
		d.expressInterest(seeder)

	case Choke:

		seeder.PeerChoking = true

	case Unchoke:

		seeder.PeerChoking = false
		pieceIndex, blockIndex, _ := d.requestPiece(seeder)
		index, offset, length := d.convertPieceIndexToOffset(pieceIndex, blockIndex)
		payload := makeRequestPayload(index, offset, length)
		seeder.outcoming <- Message{Request, payload, d.PeerId}

	case Interested:

		seeder.PeerInterested = true
		if true { // todo condition
			seeder.AmChoking = false
			seeder.outcoming <- Message{Unchoke, nil, d.PeerId}
		}

	case NotInterested:

		seeder.PeerInterested = false
		seeder.AmChoking = true
		seeder.outcoming <- Message{Choke, nil, d.PeerId}

	case Request:

		if seeder.AmChoking == false {
			//todo
		}

	case Piece:

		index, offset, data, err := parsePiecePayload(message.Payload) // todo err
		if err != nil {
			seeder.Close()
		}

		pieceIndex, blockIndex := d.convertOffsetToPieceIndex(index, offset)
		d.acceptPiece(pieceIndex, blockIndex, data)
		pieceIndex, blockIndex, interested := d.requestPiece(seeder)

		if interested {
			index, offset, length := d.convertPieceIndexToOffset(pieceIndex, blockIndex)
			payload := makeRequestPayload(index, offset, length)
			seeder.outcoming <- Message{Request, payload, d.PeerId}
		} else {
			seeder.outcoming <- Message{NotInterested, nil, d.PeerId}
			d.interestingPeerCount -= 1

			if d.interestingPeerCount == 0 && d.State.left != 0 {
				for _, seeder := range d.getSeederSlice() {
					d.expressInterest(seeder)
				}
			}
		}
	}
}

func (d *Download) handleRoutine() {

	d.initDownload()

	blocksPerPiece := d.Metadata.Info.PieceLength / int64(blockLength)

	log.Printf("Start download: piece length = %d, block length = %d, blocks per piece = %d",
		d.Metadata.Info.PieceLength, blockLength, blocksPerPiece)

	for !d.stop {

		select {

		case message := <-d.messages:
			d.handleMessage(&message)

			//default:
			//
			//	continue

		}
	}

}

func (d *Download) Start() {

	d.TrackerConnection.Start()

	for i := range d.TrackerConnection.Seeders {
		log.Printf(d.TrackerConnection.Seeders[i])

		addr, err := net.ResolveTCPAddr("tcp", d.TrackerConnection.Seeders[i])
		if err != nil {
			log.Println(err)
			continue
		}

		conn, err := net.DialTimeout(addr.Network(), addr.String(), time.Second)
		if err != nil {
			log.Println(err)
			continue
		}

		s, err := NewSeeder(d.Metadata.Info.HashSHA1, d.PeerId, d.messages)
		if err != nil {
			log.Println(err)
			continue
		}

		s.PeerBitfield = bitfield.NewBitfield(uint(d.blockCount))

		err = s.Init(conn)
		if err != nil {
			log.Println(err)
			continue
		}

		d.addSeeder(s)
	}

	go d.handleRoutine()
	//go d.manageRoutine()

	//time.Sleep(120 * time.Second)

}

func (d *Download) Stop() {

	d.TrackerConnection.Stop()

}

func createFiles(d *Download) (files []*os.File, err error) {

	basePath := d.DownloadPath

	for _, fileInfo := range d.Metadata.Info.Files {
		filePath := path.Join(basePath, d.Metadata.Info.Name)

		for _, pathPart := range fileInfo.Path {
			filePath = path.Join(filePath, pathPart)
		}

		// ignore all errors
		_ = os.Mkdir(path.Dir(filePath), 0777)

		file, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}

		err = file.Truncate(fileInfo.Length)
		if err != nil {
			return nil, err
		}

		d.Files = append(d.Files, file)

		log.Printf("File \"%s\" is created: %d bytes",
			filePath, fileInfo.Length)
	}

	return d.Files, nil

}

func NewDownload(torrentFilePath string, downloadPath string) (download *Download, err error) {

	log.Printf("Create new download from %s", torrentFilePath)

	download = new(Download)
	download.Metadata, err = ReadMetadata(torrentFilePath)
	if err != nil {
		return nil, err
	}

	if download.Metadata.Info.PieceLength%int64(blockLength) != 0 {
		panic("Unexpected piece length")
	}

	download.DownloadPath = downloadPath
	download.State.left = uint64(download.Metadata.Info.TotalLength)

	files, err := createFiles(download)
	download.overlay, err = fileoverlay.NewFileOverlay(files)

	if err != nil {
		return nil, err
	}

	download.PeerId = make([]byte, 20)
	_, err = rand.Read(download.PeerId)
	if err != nil {
		panic(err)
	}

	download.TrackerConnection, err =
		NewTrackerConnection(
			download.Metadata.Announce, download.PeerId,
			download.Metadata.Info.HashSHA1, download.Port,
			&download.State)

	if err != nil {
		return nil, err
	}

	download.seedersMap = make(map[string]*Seeder)
	download.messages = make(chan Message, 32)

	download.blocksPerPiece = uint8(download.Metadata.Info.PieceLength / int64(blockLength))
	download.pieceCount = download.Metadata.Info.PieceCount

	download.lastRequestedBlock = make(map[string]uint64)

	download.lastPieceLength = download.Metadata.Info.TotalLength % download.Metadata.Info.PieceLength
	download.lastBlockLength = download.lastPieceLength % int64(blockLength)

	download.blocksPerLastPiece = uint8(download.lastPieceLength / int64(blockLength))
	if download.lastPieceLength%int64(blockLength) > 0 {
		download.blocksPerLastPiece += 1
	}

	download.blockCount = (download.pieceCount - 1) * int64(download.blocksPerPiece)
	download.blockCount += int64(download.blocksPerLastPiece)

	download.downloadingBlockBitfield = bitfield.NewBitfield(uint(download.blockCount))
	download.downloadedBlockBitfield = bitfield.NewBitfield(uint(download.blockCount))

	download.downloadedPieceBitfield = bitfield.NewBitfield(uint(download.pieceCount))

	log.Printf("Download was created successfully: peer id = %v", download.PeerId)
	log.Println(download.blocksPerPiece)

	return download, nil
}
