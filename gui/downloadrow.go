package gui

import (
	"fmt"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/lezhenin/gotorrentclient/torrent"
	"path"
)

type DownloadRow struct {
	gtk.ListBoxRow

	download *torrent.Download

	progressBar *gtk.ProgressBar
	nameLabel   *gtk.Label
	stateLabel  *gtk.Label
	speedLabel  *gtk.Label

	lastDownloaded float64

	//sync.Mutex
}

func NewDownloadRow(download *torrent.Download) (row *DownloadRow, err error) {

	row = new(DownloadRow)

	row.download = download

	listBoxRow, err := gtk.ListBoxRowNew()
	if err != nil {
		return nil, err
	}

	row.ListBoxRow = *listBoxRow

	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}

	grid.SetRowSpacing(4)
	grid.SetColumnSpacing(8)
	grid.SetBorderWidth(8)

	row.progressBar, err = gtk.ProgressBarNew()
	if err != nil {
		return nil, err
	}

	row.progressBar.SetHExpand(true)
	row.progressBar.SetMarginBottom(0)
	row.progressBar.SetMarginTop(0)

	_, name := path.Split(download.Metadata.FileName)
	row.nameLabel, err = gtk.LabelNew(name)
	if err != nil {
		return nil, err
	}

	row.nameLabel.SetHAlign(gtk.ALIGN_START)

	row.stateLabel, err = gtk.LabelNew("Started")
	if err != nil {
		return nil, err
	}

	row.stateLabel.SetHAlign(gtk.ALIGN_END)

	row.speedLabel, err = gtk.LabelNew("0 MiB/sec")
	row.speedLabel.SetHAlign(gtk.ALIGN_END)

	grid.Attach(row.nameLabel, 0, 0, 1, 1)
	grid.Attach(row.stateLabel, 1, 0, 1, 1)
	grid.Attach(row.progressBar, 0, 1, 2, 1)
	grid.Attach(row.speedLabel, 1, 2, 1, 1)

	row.Add(grid)

	return row, nil
}

func (r *DownloadRow) onTimerTick() bool {

	total := float64(r.download.Metadata.Info.TotalLength)
	downloaded := float64(r.download.State.Downloaded())
	fraction := downloaded / total

	finished := r.download.State.Finished()
	stopped := r.download.State.Stopped()

	r.progressBar.SetFraction(fraction)

	speed := (downloaded - r.lastDownloaded) / (1024.0 * 1024.0)

	r.lastDownloaded = downloaded

	progressText := fmt.Sprintf("%.2f of %.2f MiB",
		downloaded/float64(1024*1024),
		total/float64(1024*1024))

	speedText := fmt.Sprintf("%.2f MiB/sec", speed)

	if finished || stopped {
		r.speedLabel.SetText(progressText)
	} else {
		r.speedLabel.SetText(fmt.Sprintf("%s (%s)",
			progressText, speedText))
	}

	fmt.Println("TIMER", fraction, speed)

	return !finished && !stopped

}

func (r *DownloadRow) Start() (err error) {

	if !r.download.State.Stopped() {
		return
	}

	r.stateLabel.SetText("Started")

	_, err = glib.TimeoutAdd(1000, func() bool {
		return r.onTimerTick()
	})

	if err != nil {
		return err
	}

	go func() {
		r.download.Start()
	}()

	return nil
}

func (r *DownloadRow) Stop() {

	if r.download.State.Stopped() {
		return
	}

	r.stateLabel.SetText("Stopped")

	r.download.Stop()
}
