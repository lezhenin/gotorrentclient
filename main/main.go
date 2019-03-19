package main

import (
	"github.com/lezhenin/gotorrentclient/torrent"
	"time"
)

func main() {

	//filename := "/home/iurii/Downloads/[rutor.is]Two_Steps_From_Hell_-_Dragon_2019.torrent"
	//filename := "/home/iurii/Downloads/[rutor.is]2019_-_Vladimir_Kryzh_-_JEto_vse_obo_mne.torrent"
	//filename := "/home/iurii/Downloads/[rutor.is]Kino.HD.v.2.1.8.torrent"
	//filename := "/home/iurii/Downloads/[rutor.is]Starchild_-_2019_-_Killerrobots.torrent"
	//filename := "/home/iurii/Downloads/[rutor.is]Dr._Folder_2.6.7.9.torrent"
	filename := "/home/iurii/Downloads/[rutor.is]The.Prodigy-No.Tourists.torrent"
	//filename := "/home/iurii/Downloads/[rutor.is]Black_Sabbath__13_Best_Buy_AIO_Deluxe_Edition_M.torrent"
	//_, err := metadata.ReadMetadata(filename)
	//if err != nil {
	//	panic(err)
	//}

	//fmt.Printf("%v - %T\n", data, data)

	d, err := torrent.NewDownload(filename,
		"/home/iurii/Downloads")

	if err != nil {
		panic(err)
	}

	d.Start()
	time.Sleep(3600 * time.Second)
	d.Stop()

	return

	//gtk.Init(nil)
	//
	//// Create a new toplevel window, set its title, and connect it to the
	//// "destroy" signal to exit the GTK main loop when it is destroyed.
	//win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	//if err != nil {
	//	log.Fatal("Unable to create window:", err)
	//}
	//
	//win.SetTitle("Simple Example")
	//_, _ = win.Connect("destroy", func() {
	//	gtk.MainQuit()
	//})
	//
	//box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)
	//if err != nil {
	//	log.Fatal("Unable to create box:", err)
	//}
	//
	//bar, err := gtk.ToolbarNew()
	//bar.SetStyle(gtk.TOOLBAR_ICONS)
	//
	//addImage, err := gtk.ImageNewFromIconName("list-add", 256)
	//if err != nil {
	//	log.Fatal("Unable to create image from icon name:", err)
	//}
	//
	//removeImage, err := gtk.ImageNewFromIconName("list-remove", 256)
	//if err != nil {
	//	log.Fatal("Unable to create image from icon name:", err)
	//}
	//
	//startImage, err := gtk.ImageNewFromIconName("media-playback-start", 256)
	//if err != nil {
	//	log.Fatal("Unable to create image from icon name:", err)
	//}
	//
	//stopImage, err := gtk.ImageNewFromIconName("media-playback-stop", 256)
	//if err != nil {
	//	log.Fatal("Unable to create image from icon name:", err)
	//}
	//
	//btnAdd, err := gtk.ToolButtonNew(addImage, "Add")
	//if err != nil {
	//	log.Fatal("Unable to create tool button:", err)
	//}
	//
	//btnRemove, err := gtk.ToolButtonNew(removeImage, "Remove")
	//if err != nil {
	//	log.Fatal("Unable to create tool button:", err)
	//}
	//
	//btnStart, err := gtk.ToolButtonNew(startImage, "Start")
	//if err != nil {
	//	log.Fatal("Unable to create tool button:", err)
	//}
	//
	//btnStop, err := gtk.ToolButtonNew(stopImage, "Stop")
	//if err != nil {
	//	log.Fatal("Unable to create tool button:", err)
	//}
	//
	//sep, err := gtk.SeparatorToolItemNew()
	//if err != nil {
	//	log.Fatal("Unable to create tool separator:", err)
	//}
	//
	//bar.Add(btnAdd)
	//bar.Add(btnRemove)
	//bar.Add(sep)
	//bar.Add(btnStart)
	//bar.Add(btnStop)
	//
	//box.Add(bar)
	//
	//// Create a new label widget to show in the window.
	//l, err := gtk.LabelNew("Hello, gotk3!")
	//if err != nil {
	//	log.Fatal("Unable to create label:", err)
	//}
	//
	//// Add the label to the window.
	//box.Add(l)
	//
	//win.Add(box)
	//
	//// Set the default window size.
	//win.SetDefaultSize(800, 600)
	//
	////openFile()
	////dialog, _ := gtk.FileChooserDialogNewWith1Button("Open .torrent file", nil, gtk.FILE_CHOOSER_ACTION_OPEN, "Open", gtk.RESPONSE_ACCEPT)
	////
	////dialog.GetFilename()
	//
	//// Recursively show all widgets contained in this window.
	//win.ShowAll()
	//
	//// Begin executing the GTK main loop.  This blocks until
	//// gtk.MainQuit() is run.
	//gtk.Main()
}
