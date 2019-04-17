package main

import (
	"github.com/lezhenin/gotorrentclient/internal/gtkgui"
	"os"
)

func main() {

	gTorrent, err := gtkgui.NewGoTorrentGui()
	if err != nil {
		panic(err)
	}

	os.Exit(gTorrent.Run())

}
