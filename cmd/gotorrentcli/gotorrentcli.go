package main

import (
	"flag"
	"fmt"
	"github.com/lezhenin/gotorrentclient/pkg/torrent"
	"os"
	"os/signal"
	"sync"
)

func main() {

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	torrentFilePath := flag.String("t", "", "Path to .torrent file")
	downloadDirPath := flag.String("o", "", "Path to output directory")
	keepSeeding := flag.Bool("s", false, "Keep seeding when download finished")

	flag.Parse()

	if *torrentFilePath == "" || *downloadDirPath == "" {
		fmt.Println("Path to .torrent file or output directory is not specified")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Download %s to %s\n", *torrentFilePath, *downloadDirPath)

	metadata, err := torrent.NewMetadata(*torrentFilePath)
	if err != nil {
		panic(err)
	}

	download, err := torrent.NewDownload(metadata, *downloadDirPath)

	if err != nil {
		panic(err)
	}

	var wait sync.WaitGroup
	wait.Add(1)

	go func() {
		defer wait.Done()
		download.Start()
	}()

	for {

		select {
		case <-download.Done:
			if !*keepSeeding {
				download.Stop()
				wait.Wait()
				os.Exit(0)
			}
		case <-signals:
			download.Stop()
			wait.Wait()
			os.Exit(130)
		}
	}

}
