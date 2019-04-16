package main

import (
	"github.com/lezhenin/gotorrentclient/gui"
	"os"
)

func main() {

	gTorrent, err := gui.NewGTorrent()
	if err != nil {
		panic(err)
	}

	os.Exit(gTorrent.Run())

}
