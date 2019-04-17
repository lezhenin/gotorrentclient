package main

import (
	"fmt"
	"github.com/lezhenin/gotorrentclient/pkg/torrent"
	"os"
	"sync"
)

func main() {

	args := os.Args

	if len(args) < 3 {
		panic(args)
	}

	fmt.Println(args)

	torrentFile := os.Args[1]
	folder := os.Args[2]

	metadata, err := torrent.NewMetadata(torrentFile)
	if err != nil {
		panic(err)
	}

	download, err := torrent.NewDownload(metadata, folder)

	var wait sync.WaitGroup
	wait.Add(1)

	go func() {
		defer wait.Done()
		download.Start()
	}()

	<-download.Done
	download.Stop()

	wait.Wait()

	fmt.Println("Done")

}
