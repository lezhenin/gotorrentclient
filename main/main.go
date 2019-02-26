package main

import (
	"crypto/rand"
	"fmt"
	"github.com/lezhenin/gotorrentclient/metadata"
	"net/http"
	"net/url"
	"strconv"
)

type DownloadState int

const (
	Started   DownloadState = 0
	Continued DownloadState = 1
	Stopped   DownloadState = 2
	Completed DownloadState = 3
)

type Download struct {
	Metadata   metadata.Metadata
	PeerId     []byte
	InfoHash   []byte
	Port       int64
	Uploaded   int64
	Downloaded int64
	Left       int64
	Compat     bool
	NoPeerId   bool
	State      DownloadState
	ClientIP   string
	TrackerId  []byte
}

func MakeRequestURL(download Download) string {

	Url, err := url.Parse(download.Metadata.AnnounceList[11][0])
	if err != nil {
		panic(err)
	}

	parameters := url.Values{}
	parameters.Add("info_hash", string(download.Metadata.Info.HashSHA1))
	parameters.Add("peer_id", string(download.PeerId))
	parameters.Add("port", strconv.FormatInt(download.Port, 10))

	Url.RawQuery = parameters.Encode()

	fmt.Println(Url.String())

	//buffer := bytes.Buffer{}
	////buffer.WriteString("?info_hash=")
	////buffer.Write(download.Metadata.Info.HashSHA1)
	//buffer.WriteString("?peer_id=")
	//buffer.Write(download.PeerId)
	//buffer.WriteString("&port=")
	//buffer.WriteString(strconv.FormatInt(download.Port, 10))
	//buffer.WriteString("&uploaded=")
	//buffer.WriteString(strconv.FormatInt(download.Uploaded, 10))
	//buffer.WriteString("&downloaded=")
	//buffer.WriteString(strconv.FormatInt(download.Downloaded, 10))
	//buffer.WriteString("&left=")
	//buffer.WriteString(strconv.FormatInt(download.Left, 10))
	//
	//urlPath := download.Metadata.Announce
	////urlQuery := url.QueryEscape(buffer.String())
	//urlQuery := buffer.String()
	//
	//urlPointer := url.Parse(urlPath + "/" + urlQuery)
	//
	//fmt.Println(urlPath + "/" + urlQuery)

	return Url.String()
}

func main() {

	filename := "/home/iurii/Downloads/[rutor.is]Two_Steps_From_Hell_-_Dragon_2019.torrent"
	data, err := metadata.ReadMetadata(filename)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v - %T\n", data, data)

	peerId := make([]byte, 20)
	_, err = rand.Read(peerId)
	if err != nil {
		panic(err)
	}

	fmt.Printf("peer_id %X\n", peerId)

	download := Download{}
	download.PeerId = peerId
	download.Metadata = data

	response, err := http.Get(MakeRequestURL(download))
	fmt.Println(err)
	fmt.Println(response)

}
