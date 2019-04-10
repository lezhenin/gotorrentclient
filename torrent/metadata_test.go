package torrent

import (
	"github.com/stretchr/testify/assert"
	"github.com/zeebo/bencode"
	"io/ioutil"
	"os"
	"testing"
)

func TestNewMetadata(t *testing.T) {

	filename := "../test/DA3F.torrent"
	metadata, err := NewMetadata(filename)
	assert.NoError(t, err, "can not decode metadata")

	assert.EqualValues(t, filename, metadata.FileName, "file name doesnt match")
	assert.EqualValues(t, "udp://127.0.0.1:3515", metadata.Announce, "announce url doesnt match")
	assert.True(t, metadata.Info.MultiFile, "metadata is not multi-file")
	assert.EqualValues(t, 32*1024, metadata.Info.PieceLength, "wrong piece length")
}

func TestNewMetadata_WrongPiecesCount(t *testing.T) {

	filename := "../test/DA3F.torrent"
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
