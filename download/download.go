package download

import (
	"crypto/rand"
	"fmt"
	"github.com/lezhenin/gotorrentclient/metadata"
	"github.com/lezhenin/gotorrentclient/tracker"
	"log"
	"os"
	"path"
	"sync"
	"time"
)

type Stage int

const (
	Stopped   Stage = 0
	Started   Stage = 1
	Completed Stage = 2
)

type State struct {
	uploaded   uint64
	downloaded uint64
	left       uint64
	Stage      Stage

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
	Metadata          metadata.Metadata
	State             State
	PeerId            []byte
	InfoHash          []byte
	Port              uint16
	Compatibility     bool
	NoPeerId          bool
	ClientIP          string
	DownloadPath      string
	Files             []*os.File
	TrackerConnection *tracker.Connection
}

func (d *Download) Start() {

	d.TrackerConnection.Start()

	seeders := make([]*tracker.Seeder, 50)

	for i := range d.TrackerConnection.Seeders {
		s, err := tracker.NewSeeder(d.TrackerConnection.Seeders[i], d.Metadata.Info.HashSHA1, d.PeerId)
		if err != nil {
			fmt.Println(err)
		} else {
			//go s.Routine()
			seeders[i] = s
		}
	}

	time.Sleep(30 * time.Second)

	//err = seeder.SendHandshakeMessage()
	//if err != nil {
	//	panic(err)
	//}

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
	download.Metadata, err = metadata.ReadMetadata(torrentFilePath)
	if err != nil {
		return nil, err
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
		tracker.NewTrackerConnection(
			download.Metadata.Announce, download.PeerId,
			download.Metadata.Info.HashSHA1, download.Port,
			&download.State)

	if err != nil {
		return nil, err
	}

	log.Printf("Download was created successfully: peer id = %v", download.PeerId)

	return download, nil
}
