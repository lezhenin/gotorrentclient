package torrent

import (
	"crypto/rand"
	"log"
	"net"
	"net/url"
	"time"
)

const blockLength int = 16 * 1024

type Download struct {
	Metadata     Metadata
	PeerId       []byte
	InfoHash     []byte
	Port         uint16
	NoPeerId     bool
	ClientIP     string
	DownloadPath string

	State   *State
	manager *Manager
	tracker *Tracker
	storage *Storage

	peerStatus map[string]bool
}

func (d *Download) Start() {

	go func() {
		for {

			response := <-d.tracker.announceResponseChannel

			for _, peer := range response.Peers {

				addr, err := net.ResolveTCPAddr("tcp", peer)
				if err != nil {
					log.Println(err)
					continue
				}

				conn, err := net.DialTimeout(addr.Network(), addr.String(), time.Second)
				if err != nil {
					log.Println(err)
					continue
				}

				err = d.manager.AddSeeder(conn, false)
				if err != nil {
					log.Println(err)
					continue
				}

			}
		}

	}()

	d.tracker.Run()

	d.tracker.announceRequestChannel <- AnnounceRequest{
		Started,
		d.State.Downloaded(),
		d.State.Uploaded(),
		d.State.Left(),
		8861,
		50}

	d.manager.Start()

}

func (d *Download) Stop() {

	d.tracker.announceRequestChannel <- AnnounceRequest{
		Stopped,
		d.State.Downloaded(),
		d.State.Uploaded(),
		d.State.Left(),
		8861,
		50}

}

func NewDownload(torrentFilePath string, downloadPath string) (d *Download, err error) {

	log.Printf("Create new download from %s", torrentFilePath)

	d = new(Download)

	d.Metadata, err = ReadMetadata(torrentFilePath)
	if err != nil {
		return nil, err
	}

	if d.Metadata.Info.PieceLength%int64(blockLength) != 0 {
		panic("Unexpected piece length")
	}

	d.InfoHash = d.Metadata.Info.HashSHA1

	d.PeerId = make([]byte, 20)
	_, err = rand.Read(d.PeerId)
	if err != nil {
		panic(err)
	}

	d.State = NewState(uint64(d.Metadata.Info.TotalLength))

	d.storage, err = NewStorage(d.Metadata.Info, downloadPath)
	if err != nil {
		return nil, err
	}

	d.manager = NewManager(d.PeerId, d.InfoHash, &d.Metadata.Info, d.State, d.storage)
	if err != nil {
		return nil, err
	}

	announceUrl, _ := url.Parse(d.Metadata.Announce)
	// todo check schema

	conn, _ := net.Dial("udp", announceUrl.Host)

	d.tracker, err = NewTracker(d.PeerId, d.InfoHash, conn)
	if err != nil {
		return nil, err
	}

	return d, nil
}
