package gui

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"log"
)

type GTorrent struct {
	application *gtk.Application
}

func NewGTorrent() (gTorrent *GTorrent, err error) {

	gTorrent = new(GTorrent)

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

func (g *GTorrent) Run() (status int) {

	return g.application.Run([]string{})

}

func (g *GTorrent) onActivation() {

	log.Println("Activate")

	window, err := NewMainWindow(g.application)
	if err != nil {
		log.Fatal(err)
	}

	window.ShowAll()

}
