package gtkgui

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"log"
)

type GoTorrentGui struct {
	application *gtk.Application
}

func NewGoTorrentGui() (gTorrent *GoTorrentGui, err error) {

	gTorrent = new(GoTorrentGui)

	gTorrent.application, err = gtk.ApplicationNew("com.github.lezhenin.gtorrent", glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		return nil, err
	}

	_, err = gTorrent.application.Connect("activate", func() {
		gTorrent.onActivation()
	})
	if err != nil {
		return nil, err
	}

	return gTorrent, err

}

func (g *GoTorrentGui) Run() (status int) {

	return g.application.Run([]string{})

}

func (g *GoTorrentGui) onActivation() {

	log.Println("Activate")

	window, err := NewMainWindow(g.application)
	if err != nil {
		log.Fatal(err)
	}

	window.ShowAll()

}
