package torrent

import (
	"crypto/rand"
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
	Overlay           fileoverlay.FileOverlay

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

func (d *Download) sendRoutine() {

}

func (d *Download) manageRoutine() {

	if d.stop {
		return
	}

	interestedSeeders := []*Seeder{}

	for {

		blocksPerPiece := d.Metadata.Info.PieceLength / int64(blockLength)

		if int64(d.lastRequestedBlock) == blocksPerPiece {
			d.lastRequestedPiece += 1
			d.lastRequestedBlock = 0
		}

		pieceIndex := d.lastRequestedPiece
		blockIndex := d.lastRequestedBlock

		if int64(pieceIndex) == 100 {
			return
		}

		seeders := d.getSeederSlice()
		for _, seeder := range seeders {
			log.Println("Bitfield accessed", seeder.PeerId)
			if seeder.PeerBitfield != nil && seeder.PeerBitfield.Get(uint(pieceIndex)) == 1 {
				interestedSeeders = append(interestedSeeders, seeder)
			} else if seeder.AmInterested {
				seeder.outcoming <- Message{NotInterested, nil, d.PeerId}
			}
		}

		for _, seeder := range interestedSeeders {
			//if seeder.AmChoking == true {
			//	seeder.outcoming <- Message{Unchoke, nil, d.PeerId}
			//	seeder.AmChoking = false
			//}
			if seeder.AmInterested == false {
				log.Println("int")
				seeder.outcoming <- Message{Interested, nil, d.PeerId}
				seeder.AmInterested = true
			}
			if seeder.PeerChoking == false {
				log.Println("req")
				seeder.outcoming <- Message{Request,
					makeRequestPayload(uint32(pieceIndex),
						uint32(blocksPerPiece*int64(blockIndex)),
						uint32(blockLength)), d.PeerId}
				d.lastRequestedBlock += 1
			}
		}

		//d.stop = true

		break

	}

}

func (d *Download) initDownload() {

	d.lastRequestedPiece = 0
	d.lastDownloadedPiece = -1

	d.pieces = append(d.pieces, NewPieceStatus(d.lastRequestedPiece, d.blocksPerPiece))
	for _, s := range d.getSeederSlice() {
		d.updateSeederInterested(s)
	}
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

func (d *Download) requestNextPiece(seeder *Seeder) {

	blocksPerPiece := d.Metadata.Info.PieceLength / int64(blockLength)
	if seeder.AmInterested {
		seeder.outcoming <- Message{Request,
			makeRequestPayload(uint32(d.lastRequestedPiece),
				uint32(blocksPerPiece*int64(d.lastRequestedBlock)),
				uint32(blockLength)), d.PeerId}
		d.lastRequestedBlock += 1
	}
	if int64(d.lastRequestedBlock) == blocksPerPiece {
		d.lastRequestedBlock = 0
		d.lastRequestedPiece += 1
		for _, s := range d.getSeederSlice() {
			d.updateSeederInterested(s)
		}
	}

}

func (d *Download) handleMessage(message *Message) {

	log.Println("HANDLE", message.Id)

	seeder, _ := d.getSeeder(message.PeerId)

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
		d.requestNextPiece(seeder)

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
		d.requestNextPiece(seeder)

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

	time.Sleep(60 * time.Second)

}

func (d *Download) Stop() {

	d.TrackerConnection.Stop()

}

func (d *Download) IsFinished() {

}

func createFiles(d *Download) (err error) {

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
			return err
		}

		err = file.Truncate(fileInfo.Length)
		if err != nil {
			return err
		}

		d.Files = append(d.Files, file)

		log.Printf("File \"%s\" is created: %d bytes",
			filePath, fileInfo.Length)
	}

	return nil

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

	err = createFiles(download)

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

	return download, nil
}
