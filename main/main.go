package main

import (
	"fmt"
	"github.com/lezhenin/gotorrentclient/metadata"
)

func main() {
	fmt.Println("Hello world!")
	metadata.ReadMetadata("/home/iurii/Downloads/[rutor.is]Two_Steps_From_Hell_-_Dragon_2019.torrent")
}
