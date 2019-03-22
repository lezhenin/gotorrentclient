package main

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/lezhenin/gotorrentclient/torrent"
	"log"
	"path"
)

const (
	COLUMN_FILENAME = iota
	COLUMN_LENGTH
)

func main() {

	////filename := "/home/iurii/Downloads/[rutor.is]Two_Steps_From_Hell_-_Dragon_2019.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]2019_-_Vladimir_Kryzh_-_JEto_vse_obo_mne.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Kino.HD.v.2.1.8.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Starchild_-_2019_-_Killerrobots.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Dr._Folder_2.6.7.9.torrent"
	//filename := "/home/iurii/Downloads/[rutor.is]The.Prodigy-No.Tourists.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Black_Sabbath__13_Best_Buy_AIO_Deluxe_Edition_M.torrent"
	////_, err := metadata.ReadMetadata(filename)
	////if err != nil {
	////	panic(err)
	////}
	//
	////fmt.Printf("%v - %T\n", data, data)
	//
	//d, err := torrent.NewDownload(filename,
	//	"/home/iurii/Downloads")
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//d.Start()
	//time.Sleep(3600 * time.Second)
	//d.Stop()
	//
	//return

	gtk.Init(nil)

	// Create a new toplevel window, set its title, and connect it to the
	// "destroy" signal to exit the GTK main loop when it is destroyed.
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}

	win.SetTitle("Simple Example")
	_, _ = win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)
	if err != nil {
		log.Fatal("Unable to create box:", err)
	}

	bar, err := createToolBar()

	box.Add(bar)

	// Create a new label widget to show in the window.
	l, err := gtk.LabelNew("Hello, gotk3!")
	if err != nil {
		log.Fatal("Unable to create label:", err)
	}

	// Add the label to the window.
	box.Add(l)

	win.Add(box)

	// Set the default window size.
	win.SetDefaultSize(800, 600)

	//openFile()
	//dialog, _ := gtk.FileChooserDialogNewWith1Button("Open .torrent file", nil, gtk.FILE_CHOOSER_ACTION_OPEN, "Open", gtk.RESPONSE_ACCEPT)
	//
	//dialog.GetFilename()

	// Recursively show all widgets contained in this window.
	win.ShowAll()

	// Begin executing the GTK main loop.  This blocks until
	// gtk.MainQuit() is run.
	gtk.Main()
}

func createToolBar() (bar *gtk.Toolbar, err error) {

	bar, err = gtk.ToolbarNew()
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

	//var metadata *torrent.Metadata

	_, err = btnAdd.Connect("clicked", func() {

		dialog, _ := gtk.DialogNew()
		dialog.SetTitle("Torrent options")
		_, _ = dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)
		_, _ = dialog.AddButton("Accept", gtk.RESPONSE_ACCEPT)

		box, err := dialog.GetContentArea()
		if err != nil {
			log.Fatal(err)
		}

		box.SetBorderWidth(8)

		fileChooserLabel, err := gtk.LabelNew("Torrent file:")
		if err != nil {
			log.Fatal(err)
		}

		fileChooserLabel.SetHAlign(gtk.ALIGN_START)

		fileChooserBtn, err := gtk.FileChooserButtonNew("Torrent file", gtk.FILE_CHOOSER_ACTION_OPEN)
		if err != nil {
			log.Fatal(err)
		}

		fileChooserBtn.SetHExpand(true)

		filter, err := gtk.FileFilterNew()
		if err != nil {
			log.Fatal(err)
		}

		filter.SetName("Torrent (.torrent)")
		filter.AddPattern("*.torrent")

		fileChooserBtn.SetFilter(filter)

		folderChooserLabel, err := gtk.LabelNew("Download folder:")
		if err != nil {
			log.Fatal(err)
		}

		folderChooserLabel.SetHAlign(gtk.ALIGN_START)

		folderChooserBtn, err := gtk.FileChooserButtonNew("Download folder", gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER)
		if err != nil {
			log.Fatal(err)
		}

		folderChooserBtn.SetHExpand(true)

		view, err := gtk.TreeViewNew()
		if err != nil {
			log.Fatal(err)
		}

		view.AppendColumn(createColumn("Filename", COLUMN_FILENAME))
		view.AppendColumn(createColumn("Length (bytes)", COLUMN_LENGTH))

		store, err := gtk.TreeStoreNew(glib.TYPE_STRING, glib.TYPE_INT64)
		if err != nil {
			log.Fatal(err)
		}

		view.SetModel(store)

		view.SetHExpand(true)
		view.SetVExpand(true)

		scrolledWindow, err := gtk.ScrolledWindowNew(nil, nil)
		if err != nil {
			log.Fatal(err)
		}

		scrolledWindow.Add(view)

		grid, err := gtk.GridNew()
		if err != nil {
			log.Fatal(err)
		}

		grid.SetColumnSpacing(8)
		grid.SetRowSpacing(8)

		grid.Attach(fileChooserLabel, 0, 0, 1, 1)
		grid.Attach(fileChooserBtn, 1, 0, 1, 1)
		grid.Attach(folderChooserLabel, 0, 1, 1, 1)
		grid.Attach(folderChooserBtn, 1, 1, 1, 1)
		grid.Attach(scrolledWindow, 0, 2, 2, 1)
		grid.SetHExpand(true)
		grid.SetVExpand(true)

		box.Add(grid)

		_, _ = fileChooserBtn.Connect("file-set", func(button *gtk.FileChooserButton) {
			metadata, err := torrent.ReadMetadata(button.GetFilename())
			if err != nil {
				log.Fatal(err)
			}

			iter := addRow(store, metadata.Info.Name, metadata.Info.TotalLength)

			iters := make(map[string]*gtk.TreeIter)
			lengths := make(map[string]int64)

			for _, fileInfo := range metadata.Info.Files {

				fullPath := ""
				parentIter := iter

				for _, pathPart := range fileInfo.Path {

					fullPath = path.Join(fullPath, pathPart)

					lengths[fullPath] = lengths[fullPath] + fileInfo.Length

					if _, ok := iters[fullPath]; !ok {
						iters[fullPath] = addSubRow(store, parentIter, pathPart, lengths[fullPath])
					}

					parentIter = iters[fullPath]
				}
			}

		})

		dialog.ShowAll()
		dialog.Run()

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

func createTorrentOpenDialog(parent *gtk.Window) (dialog *gtk.FileChooserDialog, err error) {

	dialog, err =
		gtk.FileChooserDialogNewWith1Button(
			"Open .torrent file", parent, gtk.FILE_CHOOSER_ACTION_OPEN,
			"Open", gtk.RESPONSE_ACCEPT)

	if err != nil {
		return nil, err
	}

	filter, _ := gtk.FileFilterNew()
	filter.SetName("Torrent (.torrent)")
	filter.AddPattern("*.torrent")

	dialog.AddFilter(filter)

	return dialog, nil
}

func createTorrentOptionDialog() (dialog *gtk.Dialog, err error) {

	dialog, err = gtk.DialogNew()
	return nil, nil

}

func createColumn(title string, id int) *gtk.TreeViewColumn {

	cellRenderer, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal(err)
	}

	column, err := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)
	if err != nil {
		log.Fatal(err)
	}

	return column
}

func addRow(treeStore *gtk.TreeStore, filename string, length int64) *gtk.TreeIter {
	return addSubRow(treeStore, nil, filename, length)
}

func addSubRow(treeStore *gtk.TreeStore, parentIter *gtk.TreeIter, filename string, length int64) *gtk.TreeIter {

	iter := treeStore.Append(parentIter)

	err := treeStore.SetValue(iter, COLUMN_FILENAME, filename)
	if err != nil {
		log.Fatal(err)
	}
	err = treeStore.SetValue(iter, COLUMN_LENGTH, length)
	if err != nil {
		log.Fatal(err)
	}

	return iter
}
