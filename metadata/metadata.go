package metadata

import (
	"crypto/sha1"
	"fmt"
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
	PieceLength int64
	Pieces      []byte
	Private     bool
	MultiFile   bool
	Files       []FileInfo
	SumSHA1     []byte
}

type Metadata struct {
	Info         Info
	Announce     string
	AnnounceList [][]string
	CreationDate time.Time
	Comment      string
	CreatedBy    string
	Encoding     string
}

type DecodeError struct {
	Source    interface{}
	FieldName string
}

func (e DecodeError) Error() string {
	return fmt.Sprintf("Can't decode field '%s': %s", e.FieldName, e.Source)
}

type RequiredFieldError struct {
	Source    interface{}
	FieldName string
}

func (e RequiredFieldError) Error() string {
	return fmt.Sprintf("Mandatory field '%s' is absent: %s", e.FieldName, e.Source)
}

type dictionary map[string]interface{}
type list []interface{}

//func getListSlice(dict dictionary, key string) (value []list, err error) {
//
//	item, ok := dict[key]; if ok {
//		item, ok := item.(list); if ok {
//			for
//		} else {
//			return list{}, DecodeError{dict[key], key}
//		}
//	} else {
//		return list{}, RequiredFieldError{dict, key}
//	}
//
//	return value, nil
//}

func getList(dict dictionary, key string) (value list, err error) {

	item, ok := dict[key]
	if ok {
		item, ok := item.(list)
		if ok {
			value = item
		} else {
			return list{}, DecodeError{dict[key], key}
		}
	} else {
		return list{}, RequiredFieldError{dict, key}
	}

	return value, nil
}

func getDict(dict dictionary, key string) (value dictionary, err error) {

	item, ok := dict[key]
	if ok {
		item, ok := item.(dictionary)
		if ok {
			value = item
		} else {
			return dictionary{}, DecodeError{dict[key], key}
		}
	} else {
		return dictionary{}, RequiredFieldError{dict, key}
	}

	return value, nil
}

func getString(dict dictionary, key string) (value string, err error) {

	item, ok := dict[key]
	if ok {
		item, ok := item.(string)
		if ok {
			value = item
		} else {
			return "", DecodeError{dict[key], key}
		}
	} else {
		return "", RequiredFieldError{dict, key}
	}

	return value, nil
}

func getInt(dict dictionary, key string) (value int64, err error) {

	item, ok := dict[key]
	if ok {
		item, ok := item.(int64)
		if ok {
			value = item
		} else {
			return 0, DecodeError{dict[key], key}
		}
	} else {
		return 0, RequiredFieldError{dict, key}
	}

	return value, nil
}

func infoDictToStruct(infoDict map[string]interface{}) (info Info, err error) {

	info = Info{}

	data, err := bencode.EncodeBytes(infoDict)
	if err != nil {
		panic(err)
	}

	hash := sha1.New()
	info.SumSHA1 = hash.Sum(data)

	pieceLength, ok := infoDict["piece length"]
	if ok {
		pieceLength, ok := pieceLength.(int64)
		if ok {
			info.PieceLength = pieceLength
		} else {
			return Info{}, DecodeError{infoDict["piece length"], "piece length"}
		}
	} else {
		return Info{}, RequiredFieldError{infoDict, "piece length"}
	}

	pieces, ok := infoDict["pieces"]
	if ok {
		pieces, ok := pieces.(string)
		if ok {
			info.Pieces = []byte(pieces)
		} else {
			return Info{}, DecodeError{infoDict["pieces"], "pieces"}
		}
	} else {
		return Info{}, RequiredFieldError{infoDict, "pieces"}
	}

	return info, nil
}

func metadataDictToStruct(metadataDict dictionary) (metadata Metadata, err error) {

	metadata = Metadata{}

	infoDict, err := getDict(metadataDict, "info")
	if err != nil {
		return Metadata{}, err
	}

	metadata.Info, err = infoDictToStruct(infoDict)
	if err != nil {
		return Metadata{}, err
	}

	metadata.Announce, err = getString(metadataDict, "announce")
	if err != nil {
		return Metadata{}, err
	}

	//announceList, err := getList(metadataDict, "announce-list")
	//for outerIndex, innerList := range announceList {
	//	innerList, ok := innerList.([]interface{})
	//}
	//
	//TODO try to simplify

	announceList, ok := metadataDict["announce-list"]
	if ok {
		announceList, ok := announceList.([]interface{})
		if ok {
			metadata.AnnounceList = make([][]string, len(announceList))
			for outerIndex, innerList := range announceList {
				innerList, ok := innerList.([]interface{})
				if ok {
					metadata.AnnounceList[outerIndex] = make([]string, len(innerList))
					for innerIndex := range innerList {
						value, ok := innerList[innerIndex].(string)
						if ok {
							metadata.AnnounceList[outerIndex][innerIndex] = value
						}
					}
				}

			}
		}
	}

	creationDate, err := getInt(metadataDict, "creation date")
	metadata.CreationDate = time.Unix(creationDate, 0)

	metadata.Encoding, err = getString(metadataDict, "encoding")

	metadata.CreatedBy, err = getString(metadataDict, "created by")

	metadata.Comment, err = getString(metadataDict, "comment")

	return metadata, nil
}

func ReadMetadata(filename string) Metadata {

	data, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	var bencodedData interface{}

	err = bencode.DecodeBytes(data, &bencodedData)

	if err != nil {
		panic(err)
	}

	metadataDict, ok := bencodedData.(map[string]interface{})

	if !ok {
		panic(ok)
	}

	metadata, err := metadataDictToStruct(metadataDict)

	fmt.Printf("err: %v, value: %v - %T", err, metadata, metadata)

	return metadata
	//return metadataFromDict()

}
