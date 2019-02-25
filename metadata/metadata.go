package metadata

import (
	"github.com/zeebo/bencode"
	"io/ioutil"
	"time"
)

type FileInfo struct {
	Length  int
	Path    string
	HashMD5 string
}

type Info struct {
	PieceLength int
	Pieces      string
	Private     bool
	Files       []FileInfo
}

type Metadata struct {
	Info         Info
	Announce     string
	AnnounceList []string
	CreationDate time.Time
	Comment      string
	CreatedBy    string
	Encoding     string
}

//func fileInfoFromDict() FileInfo {}
//func infoFromDict() Info {}
//func metadataFromDict() Metadata {
//	return Metadata{}
//}

func ReadMetada(filename string) Metadata {

	data, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	var metadata interface{}
	err = bencode.DecodeBytes(data, metadata)

	if err != nil {
		panic(err)
	}

	return Metadata{}
	//return metadataFromDict()

}
