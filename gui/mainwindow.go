package gui

import (
	"fmt"
	"github.com/gotk3/gotk3/gtk"
	"github.com/lezhenin/gotorrentclient/torrent"
	"log"
)

type MainWindow struct {
	gtk.ApplicationWindow

	listBox *gtk.ListBox
}

func NewMainWindow(application *gtk.Application) (window *MainWindow, err error) {

	window = new(MainWindow)

	baseWindow, err := gtk.ApplicationWindowNew(application)
	if err != nil {
		return nil, err
	}

	window.ApplicationWindow = *baseWindow

	window.SetTitle("gTorrent")
	window.SetDefaultSize(600, 400)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)
	if err != nil {
		log.Fatal(err)
	}

	bar, err := window.createToolBar()
	if err != nil {
		log.Fatal(err)
	}

	window.listBox, err = gtk.ListBoxNew()
	if err != nil {
		log.Fatal(err)
	}

	box.Add(bar)
	box.Add(window.listBox)

	window.Add(box)

	return window, nil
}

func (w *MainWindow) createToolBar() (bar *gtk.Toolbar, err error) {

	bar, err = gtk.ToolbarNew()
	if err != nil {
		return nil, err
	}

	bar.SetStyle(gtk.TOOLBAR_ICONS)

	addImage, err := gtk.ImageNewFromIconName("list-add", 256)
	if err != nil {
		return nil, err
	}

	removeImage, err := gtk.ImageNewFromIconName("list-remove", 256)
	if err != nil {
		return nil, err
	}

	startImage, err := gtk.ImageNewFromIconName("media-playback-start", 256)
	if err != nil {
		return nil, err
	}

	stopImage, err := gtk.ImageNewFromIconName("media-playback-stop", 256)
	if err != nil {
		return nil, err
	}

	btnAdd, err := gtk.ToolButtonNew(addImage, "Add")
	if err != nil {
		return nil, err
	}

	btnRemove, err := gtk.ToolButtonNew(removeImage, "Remove")
	if err != nil {
		return nil, err
	}

	btnStart, err := gtk.ToolButtonNew(startImage, "Start")
	if err != nil {
		return nil, err
	}

	btnStop, err := gtk.ToolButtonNew(stopImage, "Stop")
	if err != nil {
		return nil, err
	}

	sep, err := gtk.SeparatorToolItemNew()
	if err != nil {
		return nil, err
	}

	_, err = btnAdd.Connect("clicked", func() {
		w.onAddClicked()
	})

	if err != nil {
		log.Fatal(err)
	}

	bar.Add(btnAdd)
	bar.Add(btnRemove)
	bar.Add(sep)
	bar.Add(btnStart)
	bar.Add(btnStop)

	return bar, nil
}

func (w *MainWindow) onAddClicked() {

	dialog, err := NewAddDialog()
	if err != nil {
		log.Fatal(err)
	}

	dialog.SetDefaultSize(400, 400)
	dialog.ShowAll()

	response := dialog.Run()

	for !(response == gtk.RESPONSE_ACCEPT && dialog.IsDataSet()) {
		if response == gtk.RESPONSE_CANCEL {
			dialog.Close()
			return
		}
		response = dialog.Run()
	}

	metadata, downloadPath := dialog.GetData()

	dialog.Close()

	fmt.Println(metadata.FileName)

	download, err := torrent.NewDownload(metadata, downloadPath)
	if err != nil {
		log.Fatal(err)
	}

	downloadRow, err := NewDownloadRow(download)
	w.listBox.Add(downloadRow)
	w.listBox.ShowAll()

	go func() {
		err = downloadRow.Start()
		log.Println(err)
	}()
}
