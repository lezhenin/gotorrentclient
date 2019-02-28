package main

import (
	"crypto/rand"
	"fmt"
	"github.com/lezhenin/gotorrentclient/metadata"
)

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

	tracker := Tracker{Announce: data.Announce}
	trackerConnection := Connection{Tracker: tracker, Download: download}
	err = trackerConnection.EstablishConnection()

}
