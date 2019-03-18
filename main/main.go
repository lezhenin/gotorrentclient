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
}
