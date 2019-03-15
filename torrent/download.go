package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
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

type Task struct {
	PieceIndex uint
	BlockIndex uint
	PeerId     []byte
}

type PieceStatus struct {
	pieceIndex       int
	leftBlocks       []int
	downloadedBlocks []int
	interestingPeers []string
}

func NewPieceStatus(index, blocksPerPiece int) (p *PieceStatus) {

	p = new(PieceStatus)
	p.pieceIndex = index
	p.leftBlocks = make([]int, blocksPerPiece)
	for i := 0; i < blocksPerPiece; i++ {
		p.leftBlocks[i] = i
	}

	return p
}

//func (p *PieceStatus) updateInterestedPeers(seeders []*Seeder) {
//	for _, seeder := range seeders {
//		if seeder.PeerBitfield != nil && seeder.PeerBitfield.Get(uint(p.pieceIndex)) == 1 {
//			if
//		}
//	}
//
//}
//
//func (p *PieceStatus) isInterested(s *Seeder) {
//
//
//}
//

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
	TrackerConnection *Connection
	Overlay           *fileoverlay.FileOverlay

	PieceBitfield *bitfield.Bitfield
	BlockBitfield *bitfield.Bitfield

	startedTasks []Task
	failedTasks  []Task

	pieceLeft []uint

	seedersMap map[string]*Seeder
	mapMutex   sync.RWMutex

	messages chan Message
	errors   chan error

	lastDownloadedPiece int
	lastRequestedPiece  int
	lastRequestedBlock  int

	pieces []*PieceStatus

	interestedPeerIds [][]byte
	stop              bool

	blocksPerPiece int
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

func (d *Download) initDownload() {

	d.lastRequestedPiece = 0
	d.lastDownloadedPiece = -1

	d.pieces = append(d.pieces, NewPieceStatus(d.lastRequestedPiece, d.blocksPerPiece))
	for _, s := range d.getSeederSlice() {
		d.updateSeederInterested(s)
	}
}

func (d *Download) getPiece(pieceIndex int) (index int, piece *PieceStatus) {
	for index, piece := range d.pieces {
		if piece.pieceIndex == pieceIndex {
			return index, piece
		}
	}

	return -1, nil
}

func checkInterest(s *Seeder, p *PieceStatus) (interested bool) {
	for _, peerId := range p.interestingPeers {
		if peerId == string(s.PeerId) {
			return true
		}
	}
	return false
}

func (d *Download) updateSeederInterested(s *Seeder) {

	id := string(s.PeerId)
	interested := false

	for _, p := range d.pieces {

		contains := false
		for _, ip := range p.interestingPeers {
			if ip == id {
				contains = true
				interested = true
				break
			}
		}

		if !contains && s.PeerBitfield != nil && s.PeerBitfield.Get(uint(p.pieceIndex)) == 1 {
			p.interestingPeers = append(p.interestingPeers, id)
			interested = true
		}
	}

	if interested == true && s.AmInterested == false {
		s.AmInterested = true
		s.outcoming <- Message{Interested, nil, d.PeerId}
	}

	if interested == false && s.AmInterested == true {
		s.AmInterested = false
		s.outcoming <- Message{NotInterested, nil, d.PeerId}
	}

}

func (d *Download) requestPiece(seeder *Seeder) (pieceIndex, blockIndex int, stillInterested bool) {

	var piece *PieceStatus
	piece = nil
	for _, p := range d.pieces {
		if len(p.leftBlocks) > 0 && checkInterest(seeder, p) {
			piece = p
			break
		}
	}

	if piece != nil {

		pieceIndex = piece.pieceIndex
		blockIndex = piece.leftBlocks[0]
		piece.leftBlocks = piece.leftBlocks[1:]

		log.Printf("Request block %d of piece %d", blockIndex, pieceIndex)

		return pieceIndex, blockIndex, true

	} else if !d.stop {

		log.Printf("NON STOP")

		d.lastRequestedPiece += 1
		nextPieceIndex := d.lastRequestedPiece

		if int64(nextPieceIndex) < d.Metadata.Info.PieceCount {

			piece = NewPieceStatus(nextPieceIndex, d.blocksPerPiece)
			d.pieces = append(d.pieces, piece)
			for _, s := range d.getSeederSlice() {
				d.updateSeederInterested(s)
			}

			if checkInterest(seeder, piece) {

				pieceIndex = piece.pieceIndex
				blockIndex = piece.leftBlocks[0]
				piece.leftBlocks = piece.leftBlocks[1:]

				log.Printf("Request block %d of piece %d", blockIndex, pieceIndex)

				return pieceIndex, blockIndex, true

			}
		} else {
			log.Printf("STOP")
			d.stop = true
		}
	}

	return 0, 0, false

}

func (d *Download) acceptPiece(pieceIndex, blockIndex int, data []byte) {

	log.Printf("Accept block %d of piece %d", blockIndex, pieceIndex)

	index, piece := d.getPiece(int(pieceIndex))
	if piece == nil {
		panic("piece not found")
	}

	piece.downloadedBlocks = append(piece.downloadedBlocks, blockIndex)

	pieceLength := d.Metadata.Info.PieceLength
	offset := int64(pieceIndex)*pieceLength + int64(blockIndex*blockLength)
	n, err := d.Overlay.WriteAt(data, offset)
	if n != len(data) || err != nil {
		panic(err)
	}

	if len(piece.downloadedBlocks) == d.blocksPerPiece {
		log.Printf("Accept piece %d", pieceIndex)

		pieceData := make([]byte, pieceLength)
		n, err = d.Overlay.ReadAt(pieceData, pieceLength*int64(pieceIndex))
		if err != nil {
			panic(err)
		}

		hashSum := d.Metadata.Info.Pieces[pieceIndex*20 : pieceIndex*20+20]

		hash := sha1.New()
		hash.Write(pieceData)
		sum := hash.Sum(nil)

		if bytes.Compare(sum, hashSum) != 0 {
			fmt.Println("Hashes are not math!")
			//todo handle
			piece.leftBlocks = piece.downloadedBlocks

		} else {

			log.Printf("UPDATE")
			n := copy(d.pieces[index:], d.pieces[index+1:])
			d.pieces = d.pieces[:index+n]

		}
	}
}

func (d *Download) handleMessage(message *Message) {

	log.Println("HANDLE", message.Id)

	seeder, ok := d.getSeeder(message.PeerId)

	if !ok {
		log.Printf("Peer with id %v is not found. Ignore message.")
		return
	}

	switch message.Id {

	case Error:
		d.deleteSeeder(message.PeerId)

	case Bitfield:
		seeder.PeerBitfield, _ = bitfield.NewBitfieldFromBytes(message.Payload, uint(d.Metadata.Info.PieceCount))
		d.updateSeederInterested(seeder)

	case Have:
		pieceIndex, err := parseHavePayload(message.Payload)
		if err != nil {
			panic(err)
		}
		if seeder.PeerBitfield != nil {
			seeder.PeerBitfield.Set(uint(pieceIndex))
			d.updateSeederInterested(seeder)
		}

	case Choke:
		seeder.PeerChoking = true

	case Unchoke:
		seeder.PeerChoking = false

		pieceIndex, blockIndex, _ := d.requestPiece(seeder)

		payload := makeRequestPayload(uint32(pieceIndex), uint32(blockIndex*blockLength), uint32(blockLength))
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
		//todo

		log.Printf("Piecet payload len %d", len(message.Payload))

		index, offset, data, _ := parsePiecePayload(message.Payload) // todo err

		pieceIndex := int(index)
		blockIndex := int(offset) / blockLength

		d.acceptPiece(pieceIndex, blockIndex, data)
		pieceIndex, blockIndex, interested := d.requestPiece(seeder)

		if interested {
			payload := makeRequestPayload(uint32(pieceIndex), uint32(blockIndex*blockLength), uint32(blockLength))
			seeder.outcoming <- Message{Request, payload, d.PeerId}
		} else {
			seeder.outcoming <- Message{NotInterested, nil, d.PeerId}
		}

	}
}

func (d *Download) handleRoutine() {

	d.initDownload()

	blocksPerPiece := d.Metadata.Info.PieceLength / int64(blockLength)

	log.Printf("Start download: piece length = %d, block length = %d, blocks per piece = %d",
		d.Metadata.Info.PieceLength, blockLength, blocksPerPiece)

	for {
		select {
		case message := <-d.messages:
			d.handleMessage(&message)
		default:
			continue

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

		err = s.Init(conn)
		if err != nil {
			log.Println(err)
			continue
		}

		d.addSeeder(s)
	}

	go d.handleRoutine()
	//go d.manageRoutine()

	time.Sleep(120 * time.Second)

}

func (d *Download) Stop() {

	d.TrackerConnection.Stop()

}

func (d *Download) IsFinished() {

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
	download.Overlay, err = fileoverlay.NewFileOverlay(files)

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

	download.startedTasks = make([]Task, 0)
	download.failedTasks = make([]Task, 0)

	download.blocksPerPiece = int(download.Metadata.Info.PieceLength / int64(blockLength))

	blocksPerPiece := download.Metadata.Info.PieceLength / int64(blockLength)
	download.BlockBitfield = bitfield.NewBitfield(uint(download.Metadata.Info.PieceCount) * uint(blocksPerPiece))
	download.PieceBitfield = bitfield.NewBitfield(uint(download.Metadata.Info.PieceCount))

	log.Printf("Download was created successfully: peer id = %v", download.PeerId)
	log.Println(download.blocksPerPiece)

	return download, nil
}
