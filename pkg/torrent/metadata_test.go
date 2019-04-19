package torrent

import (
	"github.com/stretchr/testify/assert"
	"github.com/zeebo/bencode"
	"io/ioutil"
	"os"
	"testing"
)

func TestMetadata_New_MultiFile(t *testing.T) {

	filename := "../../test/test_download/test_data_multi_file.torrent"
	metadata, err := NewMetadata(filename)
	assert.NoError(t, err, "can not decode metadata")

	assert.EqualValues(t, filename, metadata.FileName, "file name doesnt match")
	assert.EqualValues(t, "udp://198.51.100.5:8000", metadata.Announce, "announce url doesnt match")
	assert.Len(t, metadata.AnnounceList, 1, "announce outer list len doesnt match")
	assert.Len(t, metadata.AnnounceList[0], 2, "announce inner list len doesnt match")
	assert.EqualValues(t, "udp://198.51.100.5:8000", metadata.AnnounceList[0][0], "announce in list doesnt match")
	assert.EqualValues(t, "http://198.51.100.6/announce", metadata.AnnounceList[0][1], "announce in list doesnt match")
	assert.EqualValues(t, "a comment", metadata.Comment, "comment doesnt match")
	assert.True(t, metadata.Info.MultiFile, "metadata is not multi-file")
	assert.EqualValues(t, 32*1024, metadata.Info.PieceLength, "piece length doesnt match")
	assert.EqualValues(t, 3*1024*1024, metadata.Info.TotalLength, "total length doesnt match")
}

func TestMetadata_New_SingleFile(t *testing.T) {

	filename := "../../test/test_download/test_data_single_file.torrent"
	metadata, err := NewMetadata(filename)
	assert.NoError(t, err, "can not decode metadata")

	assert.EqualValues(t, filename, metadata.FileName, "file name doesnt match")
	assert.EqualValues(t, "udp://198.51.100.5:8000", metadata.Announce, "announce url doesnt match")
	assert.Len(t, metadata.AnnounceList, 1, "announce outer list len doesnt match")
	assert.Len(t, metadata.AnnounceList[0], 2, "announce inner list len doesnt match")
	assert.EqualValues(t, "udp://198.51.100.5:8000", metadata.AnnounceList[0][0], "announce in list doesnt match")
	assert.EqualValues(t, "http://198.51.100.6/announce", metadata.AnnounceList[0][1], "announce in list doesnt match")
	assert.EqualValues(t, "a comment", metadata.Comment, "comment doesnt match")
	assert.False(t, metadata.Info.MultiFile, "metadata is not multi-file")
	assert.EqualValues(t, 32*1024, metadata.Info.PieceLength, "piece length doesnt match")
	assert.EqualValues(t, 1*1024*1024, metadata.Info.TotalLength, "total length doesnt match")
}

func TestMetadata_New_WrongPiecesCount(t *testing.T) {

	filename := "../../test/test_download/test_data_multi_file.torrent"
	file, err := os.Open(filename)
	assert.NoError(t, err, "can not open file")

	bytes, err := ioutil.ReadAll(file)
	assert.NoError(t, err, "can not read file")

	dict := make(map[string]interface{})
	err = bencode.DecodeBytes(bytes, &dict)
	assert.NoError(t, err, "can not decode data")

	pieces, ok := dict["info"].(map[string]interface{})["pieces"].(string)
	assert.True(t, ok, "can not decode pieces")

	dict["info"].(map[string]interface{})["pieces"] = pieces[1:]

	bytes, err = bencode.EncodeBytes(dict)

	file, err = ioutil.TempFile("", "TestNewMetadata_WrongPiecesCount")
	assert.NoError(t, err, "can not create file")

	_, err = file.Write(bytes)
	assert.NoError(t, err, "can not write to file")

	_, err = NewMetadata(file.Name())

	assert.Error(t, err, "metadata decoded without errors")

}
