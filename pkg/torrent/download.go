package torrent

import (
	"crypto/rand"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
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

	Done chan struct{}

	announceTimer *time.Timer

	wg sync.WaitGroup

	exit                   bool
	exitTimer              *time.Timer
	unhandledAnnounceCount int32
}

func (d *Download) Start() (err error) {

	if !d.State.Stopped() {
		return
	}

	d.wg.Add(4)

	// drain timer
	<-d.exitTimer.C

	// tracker

	announceUrl, _ := url.Parse(d.Metadata.Announce)
	conn, _ := net.Dial("udp", announceUrl.Host)

	d.tracker, err = NewTracker(d.PeerId, d.InfoHash, conn)
	if err != nil {
		err = errors.Annotate(err, "download start")
		log.WithFields(log.Fields{
			"infoHash": d.InfoHash,
		}).Error(err)
		return err
	}

	go func() {
		defer d.wg.Done()
		err = d.tracker.Run()
		if err != nil {
			err = errors.Annotate(err, "download start")
			log.WithFields(log.Fields{
				"infoHash": d.InfoHash,
			}).Error(err)
		}
	}()

	// listener

	listener, err := NewListener(8861, 8871)
	if err != nil {
		err = errors.Annotate(err, "download start")
		log.WithFields(log.Fields{
			"infoHash": d.InfoHash,
		}).Error(err)
		return err
	}

	d.ListenPort = uint16(listener.Port)

	go func() {
		defer d.wg.Done()
		err = listener.Start()
		if err != nil {
			err = errors.Annotate(err, "download start")
			log.WithFields(log.Fields{
				"infoHash": d.InfoHash,
			}).Error(err)
		}
	}()

	// manager

	go func() {
		defer d.wg.Done()
		d.manager.Start()
	}()

	// main routine

	d.exit = false
	log.Debug(d.exit)

	go func() {

		defer d.wg.Done()

		for !d.exit {

			select {
			case response := <-d.tracker.announceResponseChannel:

				atomic.AddInt32(&d.unhandledAnnounceCount, -1)

				interval := time.Duration(response.AnnounceInterval)
				d.announceTimer.Reset(time.Second * interval)

				if atomic.LoadInt32(&d.unhandledAnnounceCount) == 0 && d.State.Stopped() {
					d.exit = true
					continue
				}

				if d.State.Finished() || d.State.Stopped() {
					continue
				}

				for _, peer := range response.Peers {

					if d.State.Finished() || d.State.Stopped() {
						break
					}

					addr, err := net.ResolveTCPAddr("tcp", peer)
					if err != nil {
						err = errors.Annotate(err, "download start")
						log.WithFields(log.Fields{
							"infoHash": d.InfoHash,
						}).Error(err)
						continue
					}

					conn, err := net.DialTimeout(addr.Network(), addr.String(), time.Second)
					if err != nil {
						err = errors.Annotate(err, "download start")
						log.WithFields(log.Fields{
							"infoHash": d.InfoHash,
						}).Error(err)
						continue
					}

					err = d.manager.AddSeeder(conn, false)
					if err != nil {
						err = errors.Annotate(err, "download start")
						log.WithFields(log.Fields{
							"infoHash": d.InfoHash,
						}).Error(err)
						continue
					}

				}

			case conn := <-listener.Connections:
				log.Debug("conn accept")
				_ = d.manager.AddSeeder(conn, true)

			case <-d.manager.Done:
				log.Debug("done")
				d.State.SetFinished(true)
				d.announce(Completed, 0)
				d.Done <- struct{}{}

			case <-d.announceTimer.C:
				log.Debug("announce timer")
				d.announce(None, 50)

			case <-d.exitTimer.C:
				log.Debug("exit timer")
				d.exit = true
				d.exitTimer.Reset(0)
			}
		}

		listener.Close()
		d.tracker.Close()

	}()

	d.State.SetStopped(false)

	d.announce(Started, 100)

	d.wg.Wait()

	return nil

}

func (d *Download) Stop() {

	if d.State.Stopped() {
		return
	}

	d.manager.Stop()
	d.announce(Stopped, 0)
	d.exitTimer.Reset(time.Second * 5)

	d.State.SetStopped(true)

	d.wg.Wait()

}

func (d *Download) announce(event Event, peersCount uint32) {

	atomic.AddInt32(&d.unhandledAnnounceCount, 1)

	d.tracker.announceRequestChannel <- AnnounceRequest{
		event,
		d.State.Downloaded(),
		d.State.Uploaded(),
		d.State.Left(),
		d.ListenPort,
		peersCount,
	}

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

	d.peerStatus = make(map[string]bool)
	d.Done = make(chan struct{})

	d.announceTimer = time.NewTimer(0)
	<-d.announceTimer.C

	d.exitTimer = time.NewTimer(0)
	//<-d.exitTimer.C

	log.SetLevel(log.TraceLevel)

	return d, nil
}
