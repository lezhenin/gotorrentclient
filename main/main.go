package main

import (
	"fmt"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/lezhenin/gotorrentclient/torrent"
	"log"
	"math"
	"path"
)

const (
	COLUMN_FILENAME = iota
	COLUMN_LENGTH
)

var downloadList []*torrent.Download
var metadata *torrent.Metadata
var folder string
var fileChoosen bool
var folderChoosen bool
var listBox *gtk.ListBox

func main() {

	////filename := "/home/iurii/Downloads/[rutor.is]Two_Steps_From_Hell_-_Dragon_2019.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]2019_-_Vladimir_Kryzh_-_JEto_vse_obo_mne.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Kino.HD.v.2.1.8.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Starchild_-_2019_-_Killerrobots.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Dr._Folder_2.6.7.9.torrent"
	//filename := "/home/iurii/Downloads/[rutor.is]The.Prodigy-No.Tourists.torrent"
	////filename := "/home/iurii/Downloads/[rutor.is]Black_Sabbath__13_Best_Buy_AIO_Deluxe_Edition_M.torrent"
	////_, err := metadata.NewMetadata(filename)
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

	provider, err := gtk.CssProviderNew()
	if err != nil {
		log.Fatal("Unable to create css provider:", err)
	}

	err = provider.LoadFromData("progress, trough { min-height: 16px; }")
	if err != nil {
		log.Fatal("Unable to load from path:", err)
	}

	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		log.Fatal("Unable to get default screen:", err)
	}

	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_USER)

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

	listBox, err = gtk.ListBoxNew()
	if err != nil {
		log.Fatal("Unable to create list box:", err)
	}

	box.Add(listBox)

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
		view.AppendColumn(createColumn("Length (MiB)", COLUMN_LENGTH))

		store, err := gtk.TreeStoreNew(glib.TYPE_STRING, glib.TYPE_FLOAT)
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
			metadata, err = torrent.NewMetadata(button.GetFilename())
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

			fileChoosen = true

			//if fileChoosen && folderChoosen {
			//
			//}

		})

		_, _ = folderChooserBtn.Connect("file-set", func(button *gtk.FileChooserButton) {

			folder = button.GetFilename()

		})

		dialog.ShowAll()
		response := dialog.Run()
		if response == gtk.RESPONSE_ACCEPT {
			fmt.Println("accept")

			fmt.Println(metadata.FileName)

			download, err := torrent.NewDownload(metadata, folder)
			if err != nil {
				log.Fatal(err)
			}

			downloadList = append(downloadList, download)
			addListRow(listBox, download)
			download.Start()
			fmt.Println("START")

		}

		dialog.Close()

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
	err = treeStore.SetValue(iter, COLUMN_LENGTH, float64(length)/math.Pow(2, 20))
	if err != nil {
		log.Fatal(err)
	}

	return iter
}

func addListRow(listBox *gtk.ListBox, download *torrent.Download) {

	listRow, err := gtk.ListBoxRowNew()
	if err != nil {
		log.Fatal("Unable to create list box row:", err)
	}

	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal("Unable to create grid:", err)
	}

	grid.SetRowSpacing(4)
	grid.SetColumnSpacing(8)
	grid.SetBorderWidth(8)

	progressBar, err := gtk.ProgressBarNew()
	if err != nil {
		log.Fatal("Unable to create progress bar:", err)
	}

	progressBar.SetHExpand(true)

	progressBar.SetMarginBottom(0)
	progressBar.SetMarginTop(0)

	_, name := path.Split(download.Metadata.FileName)
	nameLabel, err := gtk.LabelNew(name)
	if err != nil {
		log.Fatal("Unable to create name label:", err)
	}

	nameLabel.SetHAlign(gtk.ALIGN_START)

	stateLabel, err := gtk.LabelNew("Started")
	if err != nil {
		log.Fatal("Unable to create state label:", err)
	}

	stateLabel.SetHAlign(gtk.ALIGN_END)

	speedLabel, err := gtk.LabelNew("0 MiB/sec")
	speedLabel.SetHAlign(gtk.ALIGN_END)

	grid.Attach(nameLabel, 0, 0, 1, 1)
	grid.Attach(stateLabel, 1, 0, 1, 1)
	grid.Attach(progressBar, 0, 1, 2, 1)
	grid.Attach(speedLabel, 1, 2, 1, 1)

	listRow.Add(grid)

	lastDownloaded := float64(0)

	_, err = glib.TimeoutAdd(1000, func() bool {

		total := float64(download.Metadata.Info.TotalLength)
		downloaded := float64(download.State.Downloaded())
		fraction := downloaded / total

		finished := math.Abs(fraction-1) < 1e-12

		progressBar.SetFraction(fraction)

		speed := (downloaded - lastDownloaded) / (1024.0 * 1024.0)
		if finished {
			speed = 0
			stateLabel.SetText("Finished")
		}

		speedLabel.SetText(
			fmt.Sprintf("%.2f of %.2f MiB (%.2f MiB/sec)",
				downloaded/float64(1024*1024),
				total/float64(1024*1024),
				speed))

		lastDownloaded = downloaded

		fmt.Println("TIMER", fraction, speed)

		return !finished
	})

	if err != nil {
		log.Fatal("Unable to create star:", err)
	}

	listBox.Insert(listRow, 0)
	fmt.Println("insert")

	listBox.ShowAll()

}
