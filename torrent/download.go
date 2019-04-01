package torrent

import (
	"crypto/rand"
	log "github.com/sirupsen/logrus"
	"net"
	"net/url"
	"sync"
	"time"
)

const blockLength int = 16 * 1024

type Download struct {
	Metadata     *Metadata
	PeerId       []byte
	InfoHash     []byte
	ListenPort   uint16
	NoPeerId     bool
	ClientIP     string
	DownloadPath string

	State   *State
	manager *Manager
	tracker *Tracker
	storage *Storage

	peerStatus map[string]bool

	Done   chan struct{}
	Errors chan error

	completed bool

	announceTimer *time.Timer

	wait sync.WaitGroup
}

func (d *Download) Start() {

	if !d.State.Stopped() {
		return
	}

	d.State.SetStopped(false)

	listener, err := NewListener(8861, 8871)
	if err != nil {
		//todo
		panic(err)
	}

	go func() {
		d.wait.Add(1)
		defer d.wait.Done()
		err = listener.Start()
		log.Println(err)
	}()

	go func() {

		d.wait.Add(1)
		defer d.wait.Done()

		for !d.State.Stopped() {

			select {
			case response := <-d.tracker.announceResponseChannel:

				interval := time.Duration(response.AnnounceInterval)
				d.announceTimer.Reset(time.Second * interval)

				if d.completed {
					continue
				}

				// todo stopSignals while connecting
				for _, peer := range response.Peers {

					if d.State.Stopped() {
						return
					}

					_, ok := d.peerStatus[peer]
					if ok {
						continue
					}

					d.peerStatus[peer] = false

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

					d.peerStatus[peer] = true

				}

				log.Printf("OUT")

			case conn := <-listener.Connections:
				_ = d.manager.AddSeeder(conn, true)

			case <-d.manager.Done:
				d.completed = true
				d.announce(Completed, 0)
				d.State.SetFinished(true)
				d.Done <- struct{}{}

			case <-d.announceTimer.C:
				d.announce(None, 50)
			}

		}

	}()

	d.tracker.Run()
	d.announce(Started, 100)

	go func() {
		d.wait.Add(1)
		defer d.wait.Done()
		d.manager.Start()
	}()

	d.wait.Wait()

}

func (d *Download) Stop() {

	if d.State.Stopped() {
		return
	}

	d.manager.Stop()
	d.announce(Stopped, 0)

	d.State.SetStopped(true)
}

func (d *Download) announce(event Event, peersCount uint32) {

	d.tracker.announceRequestChannel <- AnnounceRequest{
		event,
		d.State.Downloaded(),
		d.State.Uploaded(),
		d.State.Left(),
		d.ListenPort,
		peersCount}
}

func NewDownload(metadata *Metadata, downloadPath string) (d *Download, err error) {

	d = new(Download)

	d.Metadata = metadata

	if d.Metadata.Info.PieceLength%int64(blockLength) != 0 {
		panic("Unexpected piece length")
	}

	d.InfoHash = d.Metadata.Info.HashSHA1

	d.PeerId = make([]byte, 20)
	_, err = rand.Read(d.PeerId)
	if err != nil {
		panic(err)
	}

	d.State = NewState(uint64(d.Metadata.Info.TotalLength), uint(d.Metadata.Info.PieceCount))
	d.State.SetStopped(true)

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

	d.peerStatus = make(map[string]bool)

	d.announceTimer = time.NewTimer(0)
	<-d.announceTimer.C

	log.SetLevel(log.TraceLevel)

	return d, nil
}
