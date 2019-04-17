package torrent

import (
	"crypto/sha1"
	"fmt"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
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
	PieceCount  int64
	Pieces      []byte
	Private     bool
	MultiFile   bool
	Name        string
	Files       []FileInfo
	HashSHA1    []byte
	TotalLength int64
}

type Metadata struct {
	Info         Info
	Announce     string
	AnnounceList [][]string
	CreationDate time.Time
	Comment      string
	CreatedBy    string
	Encoding     string
	FileName     string
}

type DecodeError struct {
	Source    interface{}
	FieldName string
}

func (e DecodeError) Error() string {
	return fmt.Sprintf("can't decode field '%s': %s", e.FieldName, e.Source)
}

type FieldError struct {
	Source    interface{}
	FieldName string
}

func (e FieldError) Error() string {
	return fmt.Sprintf("field '%s' is absent: %s", e.FieldName, e.Source)
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
			return list{}, errors.Annotate(
				DecodeError{dict[key], key},
				"get list")
		}
	} else {
		return list{}, errors.Annotate(
			FieldError{dict, key},
			"get list")
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
			return dictionary{}, errors.Annotate(
				DecodeError{dict[key], key},
				"get dictionary")
		}
	} else {
		return dictionary{}, errors.Annotate(
			FieldError{dict, key},
			"get dictionary")
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
			return "", errors.Annotate(
				DecodeError{dict[key], key},
				"get string")
		}
	} else {
		return "", errors.Annotate(
			FieldError{dict, key},
			"get string")
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
			return 0, errors.Annotate(
				DecodeError{dict[key], key},
				"get int")
		}
	} else {
		return 0, errors.Annotate(
			FieldError{dict, key},
			"get int")
	}

	return value, nil
}

func infoDictToStruct(infoDict map[string]interface{}) (info Info, err error) {

	info = Info{}

	data, err := bencode.EncodeBytes(infoDict)
	if err != nil {
		return Info{}, errors.Annotate(err, "convert info dictionary to struct")
	}

	hash := sha1.New()
	hash.Write(data)
	info.HashSHA1 = hash.Sum(nil)

	info.PieceLength, err = getInt(infoDict, "piece length")
	if err != nil {
		return Info{}, errors.Annotate(err, "convert info dictionary to struct")
	}

	pieces, err := getString(infoDict, "pieces")
	if err != nil {
		return Info{}, errors.Annotate(err, "convert info dictionary to struct")
	}

	info.Pieces = []byte(pieces)
	info.PieceCount = int64(len(pieces) / 20)
	if len(pieces)%20 != 0 {
		return Info{}, errors.Annotate(errors.New("piece count is not a multiple of 20"),
			"convert info dictionary to struct")
	}

	private, err := getInt(infoDict, "private")
	if err == nil {
		info.Private = private != 0
	}

	info.Name, err = getString(infoDict, "name")
	if err != nil {
		return Info{}, errors.Annotate(err, "convert info dictionary to struct")
	}

	length, err := getInt(infoDict, "length")
	if err != nil {
		info.MultiFile = true
	}

	if info.MultiFile {
		files, err := getList(infoDict, "files")
		if err != nil {
			return Info{}, errors.Annotate(err, "convert info dictionary to struct")
		}

		for _, file := range files {
			fileDict, ok := file.(map[string]interface{})
			if ok {

				fileInfo := FileInfo{}

				pathList, err := getList(fileDict, "path")
				if err != nil {
					return Info{}, errors.Annotate(err, "convert info dictionary to struct")
				}

				for _, pathItem := range pathList {
					pathString, ok := pathItem.(string)
					if ok {
						fileInfo.Path = append(fileInfo.Path, pathString)
					} else {
						return Info{}, errors.Annotate(DecodeError{pathItem, "path"},
							"convert info dictionary to struct")
					}
				}

				fileInfo.Length, err = getInt(fileDict, "length")
				if err != nil {
					return Info{}, errors.Annotate(err, "convert info dictionary to struct")
				}

				hashMD5, err := getString(fileDict, "md5sum")
				if err == nil {
					fileInfo.HashMD5 = []byte(hashMD5)
				}

				info.Files = append(info.Files, fileInfo)
				info.TotalLength += fileInfo.Length

			}
		}

	} else {

		fileInfo := FileInfo{Length: length, Path: []string{info.Name}}
		hashMD5, err := getString(infoDict, "md5sum")
		if err == nil {
			fileInfo.HashMD5 = []byte(hashMD5)
		}
		info.Name = ""
		info.TotalLength = length
	}

	return info, nil
}

func metadataDictToStruct(metadataDict dictionary) (metadata Metadata, err error) {

	metadata = Metadata{}

	infoDict, err := getDict(metadataDict, "info")
	if err != nil {
		return Metadata{},
			errors.Annotate(err, "convert metadata dictionary to struct")
	}

	metadata.Info, err = infoDictToStruct(infoDict)
	if err != nil {
		return Metadata{},
			errors.Annotate(err, "convert metadata dictionary to struct")
	}

	metadata.Announce, err = getString(metadataDict, "announce")
	if err != nil {
		return Metadata{},
			errors.Annotate(err, "convert metadata dictionary to struct")
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

	// optional fields
	metadata.Encoding, _ = getString(metadataDict, "encoding")
	metadata.CreatedBy, _ = getString(metadataDict, "created by")
	metadata.Comment, _ = getString(metadataDict, "comment")
	creationDate, err := getInt(metadataDict, "creation date")
	if err == nil {
		metadata.CreationDate = time.Unix(creationDate, 0)
	}

	return metadata, nil
}

func NewMetadata(filename string) (metadata *Metadata, err error) {

	log.Printf("Read metadata from %s\n", filename)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Annotate(err, "new metadata")
	}

	var bencodedData interface{}

	err = bencode.DecodeBytes(data, &bencodedData)
	if err != nil {
		return nil, errors.Annotate(err, "new metadata")
	}

	metadataDict, ok := bencodedData.(map[string]interface{})
	if !ok {
		return nil,
			errors.Annotate(errors.New("root element is not dictionary"),
				"new metadata")
	}

	metadata = new(Metadata)
	*metadata, err = metadataDictToStruct(metadataDict)
	if err != nil {
		return nil, errors.Annotate(err, "new metadata")
	}

	metadata.FileName = filename

	log.Printf("Metadata was read successfully:"+
		" files count = %d, piece count = %d, total length = %d\n",
		len(metadata.Info.Files), metadata.Info.PieceCount, metadata.Info.TotalLength)

	return metadata, nil

}
