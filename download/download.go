package download

import (
	"crypto/rand"
	"github.com/lezhenin/gotorrentclient/metadata"
	"github.com/lezhenin/gotorrentclient/tracker"
	"log"
	"os"
	"path"
)

type Stage int

const (
	Stopped   Stage = 0
	Started   Stage = 1
	Completed Stage = 2
)

type State struct {
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Stage      Stage
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
	TrackerConnection *tracker.TrackerConnection
}

func (d *Download) Start() {

}

func (d *Download) Stop() {

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
	download.State.Left = uint64(download.Metadata.Info.TotalLength)

	err = createFiles(download)

	if err != nil {
		return nil, err
	}

	download.PeerId = make([]byte, 20)
	_, err = rand.Read(download.PeerId)
	if err != nil {
		panic(err)
	}

	download.TrackerConnection, err = tracker.NewTrackerConnection(&download.State, download.Metadata.Announce)
	if err != nil {
		return nil, err
	}

	log.Printf("Download was created successfully: peer id = %v", download.PeerId)

	return download, nil
}
