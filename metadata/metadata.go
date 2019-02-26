package metadata

import (
	"crypto/sha1"
	"fmt"
	"github.com/zeebo/bencode"
	"io/ioutil"
	"time"
)

type FileInfo struct {
	Length  int64
	Path    []string
	HashMD5 []byte
}

type Info struct {
	PieceLength int64
	Pieces      []byte
	Private     bool
	MultiFile   bool
	Name        string
	Files       []FileInfo
	HashSHA1    []byte
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

func getList(dict dictionary, key string) (value list, err error) {

	item, ok := dict[key]
	if ok {
		item, ok := item.([]interface{})
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
		item, ok := item.(map[string]interface{})
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
	info.HashSHA1 = hash.Sum(data)

	info.PieceLength, err = getInt(infoDict, "piece length")
	if err != nil {
		return Info{}, err
	}

	pieces, err := getString(infoDict, "pieces")
	if err != nil {
		return Info{}, err
	} else {
		info.Pieces = []byte(pieces)
	}

	private, err := getInt(infoDict, "private")
	if err == nil {
		info.Private = private != 0
	}

	info.Name, err = getString(infoDict, "name")
	if err != nil {
		return Info{}, err
	}

	length, err := getInt(infoDict, "length")
	if err != nil {
		info.MultiFile = true
	}

	if info.MultiFile {
		files, err := getList(infoDict, "files")
		if err != nil {
			return Info{}, err
		}

		for _, file := range files {
			fileDict, ok := file.(map[string]interface{})
			if ok {

				fileInfo := FileInfo{}

				pathList, err := getList(fileDict, "path")
				if err != nil {
					return Info{}, err
				}

				for _, pathItem := range pathList {
					pathString, ok := pathItem.(string)
					if ok {
						fileInfo.Path = append(fileInfo.Path, pathString)
					} else {
						return Info{}, DecodeError{pathItem, "path"}
					}
				}

				fileInfo.Length, err = getInt(fileDict, "length")
				if err != nil {
					return Info{}, err
				}

				hashMD5, err := getString(fileDict, "md5sum")
				if err == nil {
					fileInfo.HashMD5 = []byte(hashMD5)
				}
			}
		}

	} else {
		fileInfo := FileInfo{Length: length, Path: []string{info.Name}}
		hashMD5, err := getString(infoDict, "md5sum")
		if err == nil {
			fileInfo.HashMD5 = []byte(hashMD5)
		}
		info.Name = "./"
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

	announceList, err := getList(metadataDict, "announce-list")
	if err == nil {
		for _, innerList := range announceList {
			innerList, ok := innerList.([]interface{})
			if ok {
				var stringList []string
				for _, item := range innerList {
					stringItem, ok := item.(string)
					if ok {
						stringList = append(stringList, stringItem)
					}
				}
				if len(stringList) > 0 {
					metadata.AnnounceList = append(metadata.AnnounceList, stringList)
				}
			}
		}
	}

	creationDate, err := getInt(metadataDict, "creation date")
	if err == nil {
		metadata.CreationDate = time.Unix(creationDate, 0)
	}

	metadata.Encoding, err = getString(metadataDict, "encoding")

	metadata.CreatedBy, err = getString(metadataDict, "created by")

	metadata.Comment, err = getString(metadataDict, "comment")

	return metadata, nil
}

func ReadMetadata(filename string) (metadata Metadata, err error) {

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

	metadata, err = metadataDictToStruct(metadataDict)
	if err != nil {
		return Metadata{}, err
	}

	return metadata, nil

}
