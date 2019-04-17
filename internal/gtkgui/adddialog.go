package gtkgui

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/lezhenin/gotorrentclient/pkg/torrent"
	"log"
	"math"
	"path"
)

const (
	COLUMN_FILENAME = iota
	COLUMN_LENGTH
)

type AddDialog struct {
	gtk.Dialog

	metadata     *torrent.Metadata
	downloadPath string

	isMetadataLoaded  bool
	isDownloadPathSet bool

	treeStore *gtk.TreeStore
}

func NewAddDialog() (dialog *AddDialog, err error) {

	dialog = new(AddDialog)

	baseDialog, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}

	dialog.Dialog = *baseDialog

	dialog.SetTitle("Add download")

	_, err = dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	if err != nil {
		return nil, err
	}

	_, err = dialog.AddButton("Accept", gtk.RESPONSE_ACCEPT)
	if err != nil {
		return nil, err
	}

	box, err := dialog.GetContentArea()
	if err != nil {
		return nil, err
	}

	box.SetBorderWidth(8)

	fileChooserLabel, err := gtk.LabelNew("Torrent file:")
	if err != nil {
		return nil, err
	}

	fileChooserLabel.SetHAlign(gtk.ALIGN_START)

	fileChooserBtn, err := gtk.FileChooserButtonNew("Torrent file", gtk.FILE_CHOOSER_ACTION_OPEN)
	if err != nil {
		return nil, err
	}

	fileChooserBtn.SetHExpand(true)

	filter, err := gtk.FileFilterNew()
	if err != nil {
		return nil, err
	}

	filter.SetName("Torrent (.torrent)")
	filter.AddPattern("*.torrent")

	fileChooserBtn.SetFilter(filter)

	_, err = fileChooserBtn.Connect("file-set", func(button *gtk.FileChooserButton) {
		dialog.onFileSet(button)
	})

	if err != nil {
		return nil, err
	}

	folderChooserLabel, err := gtk.LabelNew("Download folder:")
	if err != nil {
		return nil, err
	}

	folderChooserLabel.SetHAlign(gtk.ALIGN_START)

	folderChooserBtn, err := gtk.FileChooserButtonNew("Download folder", gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER)
	if err != nil {
		return nil, err
	}

	folderChooserBtn.SetHExpand(true)

	_, err = folderChooserBtn.Connect("file-set", func(button *gtk.FileChooserButton) {
		dialog.onFolderSet(button)
	})

	if err != nil {
		return nil, err
	}

	view, err := gtk.TreeViewNew()
	if err != nil {
		return nil, err
	}

	view.AppendColumn(createColumn("Filename", COLUMN_FILENAME))
	view.AppendColumn(createColumn("Length (MiB)", COLUMN_LENGTH))

	dialog.treeStore, err = gtk.TreeStoreNew(glib.TYPE_STRING, glib.TYPE_FLOAT)
	if err != nil {
		log.Fatal(err)
	}

	view.SetModel(dialog.treeStore)

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

	return dialog, err

}

func (d *AddDialog) addRow(filename string, length int64) *gtk.TreeIter {
	return d.addSubRow(nil, filename, length)
}

func (d *AddDialog) addSubRow(parentIter *gtk.TreeIter, filename string, length int64) *gtk.TreeIter {

	iter := d.treeStore.Append(parentIter)

	err := d.treeStore.SetValue(iter, COLUMN_FILENAME, filename)
	if err != nil {
		log.Fatal(err)
	}
	err = d.treeStore.SetValue(iter, COLUMN_LENGTH, float64(length)/math.Pow(2, 20))
	if err != nil {
		log.Fatal(err)
	}

	return iter
}

func (d *AddDialog) onFileSet(button *gtk.FileChooserButton) {

	var err error

	d.metadata, err = torrent.NewMetadata(button.GetFilename())
	if err != nil {
		log.Fatal(err)
	}

	iter := d.addRow(d.metadata.Info.Name, d.metadata.Info.TotalLength)

	iters := make(map[string]*gtk.TreeIter)
	lengths := make(map[string]int64)

	for _, fileInfo := range d.metadata.Info.Files {

		fullPath := ""
		parentIter := iter

		for _, pathPart := range fileInfo.Path {

			fullPath = path.Join(fullPath, pathPart)

			lengths[fullPath] = lengths[fullPath] + fileInfo.Length

			if _, ok := iters[fullPath]; !ok {
				iters[fullPath] = d.addSubRow(parentIter, pathPart, lengths[fullPath])
			}

			parentIter = iters[fullPath]
		}
	}

	d.isMetadataLoaded = true

}

func (d *AddDialog) onFolderSet(button *gtk.FileChooserButton) {

	d.downloadPath = button.GetFilename()
	d.isDownloadPathSet = true
}

func (d *AddDialog) IsDataSet() bool {
	return d.isMetadataLoaded && d.isDownloadPathSet
}

func (d *AddDialog) GetData() (*torrent.Metadata, string) {
	return d.metadata, d.downloadPath
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
