package torrent

import (
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"
)

const blockLength int = 16 * 1024

type Download struct {
	Metadata     *Metadata
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

	Done   chan struct{}
	Errors chan error

	completed bool

	announceTimer *time.Timer
}

func (d *Download) Start() {

	l, err := net.Listen("tcp", ":8861")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
	} else {

		go func() {
			for {
				// Listen for an incoming connection.
				conn, err := l.Accept()
				if err != nil {
					fmt.Println("Error accepting: ", err.Error())
					continue
				}

				log.Printf("ACCEPT")

				err = d.manager.AddSeeder(conn, true)
				if err != nil {
					fmt.Println("Error accepting: ", err.Error())
					continue
				}
			}

		}()
	}

	go func() {
		for {

			select {
			case response := <-d.tracker.announceResponseChannel:

				interval := time.Duration(response.AnnounceInterval)
				d.announceTimer.Reset(time.Second * interval)

				if d.completed {
					continue
				}

				for _, peer := range response.Peers {

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

			case err := <-d.tracker.errorChannel:
				d.Errors <- err
				panic(err) //todo
				return

			case err := <-d.manager.errors:
				d.Errors <- err
				panic(err) //todo
				return

			case <-d.manager.Done:
				d.completed = true
				d.announce(Completed)
				d.Done <- struct{}{}

			case <-d.announceTimer.C:
				d.announce(None)
			}

		}

	}()

	d.tracker.Run()
	d.announce(Started)
	d.manager.Start()

}

func (d *Download) Stop() {

	d.manager.Stop()
	d.announce(Stopped)

}

func (d *Download) announce(event Event) {

	d.tracker.announceRequestChannel <- AnnounceRequest{
		event,
		d.State.Downloaded(),
		d.State.Uploaded(),
		d.State.Left(),
		8861,
		50}
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

	return d, nil
}
