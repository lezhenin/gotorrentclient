package main

import (
	"github.com/lezhenin/gotorrentclient/download"
)

func main() {

	//filename := "/home/iurii/Downloads/[rutor.is]Two_Steps_From_Hell_-_Dragon_2019.torrent"
	filename := "/home/iurii/Downloads/[rutor.is]Kino.HD.v.2.1.8.torrent"
	//_, err := metadata.ReadMetadata(filename)
	//if err != nil {
	//	panic(err)
	//}

	//fmt.Printf("%v - %T\n", data, data)

	d, err := download.NewDownload(filename,
		"/home/iurii/Documents/go/src/github.com/lezhenin/gotorrentclient/filetest")

	if err != nil {
		panic(err)
	}

	d.Start()
	//time.Sleep(30 * time.Second)
	d.Stop()
}
