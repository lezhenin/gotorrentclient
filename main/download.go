package main

import (
	"fmt"
	"github.com/lezhenin/gotorrentclient/metadata"
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
	Metadata      metadata.Metadata
	PeerId        []byte
	InfoHash      []byte
	Port          uint16
	Uploaded      uint64
	Downloaded    uint64
	Left          uint64
	Compat        bool
	NoPeerId      bool
	State         DownloadState
	ClientIP      string
	TrackerId     []byte
	TransactionId []byte
	ConnectionId  []byte
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
