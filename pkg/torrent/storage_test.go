package torrent

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
)

const fileSize = 64 * 1024
const blockSize = 16 * 1024

func makeTestInfo() (info Info, filnames []string) {

	info = Info{}

	for i := 0; i < 3; i++ {
		folder := fmt.Sprintf("folder%d", i)
		name := fmt.Sprintf("file%d", i)
		fileInfo := FileInfo{fileSize, []string{folder, name}, []byte{}}
		info.Files = append(info.Files, fileInfo)
		filnames = append(filnames, path.Join(folder, name))
	}

	return info, filnames

}

func prepareStorage(testLabel string) (storage *Storage, files []*os.File) {

	dir, err := ioutil.TempDir("", testLabel)
	if err != nil {
		panic(err)
	}

	info, filenames := makeTestInfo()

	storage, err = NewStorage(info, dir)
	if err != nil {
		panic(err)
	}

	for _, filename := range filenames {
		file, err := os.OpenFile(path.Join(dir, filename), os.O_RDWR, os.ModePerm)
		if err != nil {
			panic(err)
		}
		files = append(files, file)
	}

	return storage, files
}

func TestStorage_WriteAt_OneFile(t *testing.T) {

	storage, files := prepareStorage("TestStorage_WriteAt_OneFile")

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	n, err := storage.WriteAt(testData, fileSize/2)
	assert.NoError(t, err, "can not write to storage")
	assert.EqualValues(t, blockSize, n, "write bytes != block size")

	_, err = files[0].ReadAt(data, fileSize/2)
	assert.NoError(t, err, "can not read from file")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))

	rand.Read(testData)

	n, err = storage.WriteAt(testData, fileSize+fileSize/2)
	assert.NoError(t, err, "can not write to storage")
	assert.EqualValues(t, blockSize, n, "write bytes != block size")

	_, err = files[1].ReadAt(data, fileSize/2)
	assert.NoError(t, err, "can not read from file")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))

}

func TestStorage_WriteAt_TwoFiles(t *testing.T) {

	storage, files := prepareStorage("TestStorage_WriteAt_TwoFiles")

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	// border write

	n, err := storage.WriteAt(testData, fileSize-blockSize/2)
	assert.NoError(t, err, "can not write to storage")
	assert.EqualValues(t, blockSize, n, "write bytes != block size")

	_, err = files[0].ReadAt(data[:blockSize/2], fileSize-blockSize/2)
	assert.NoError(t, err, "can not read from file")

	_, err = files[1].ReadAt(data[blockSize/2:], 0)
	assert.NoError(t, err, "can not read from file")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))

}

func TestStorage_WriteAt_ThreeFiles(t *testing.T) {

	storage, files := prepareStorage("TestStorage_WriteAt_ThreeFiles")

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	// large block

	data = make([]byte, fileSize+blockSize)

	testData = make([]byte, fileSize+blockSize)
	rand.Read(testData)

	n, err := storage.WriteAt(testData, fileSize-blockSize/2)
	assert.NoError(t, err, "can not write to storage")
	assert.EqualValues(t, fileSize+blockSize, n, "write bytes != block size")

	_, err = files[0].ReadAt(data[:blockSize/2], fileSize-blockSize/2)
	assert.NoError(t, err, "can not read from file")

	_, err = files[1].ReadAt(data[(blockSize/2):(fileSize+blockSize/2)], 0)
	assert.NoError(t, err, "can not read from file")

	_, err = files[2].ReadAt(data[(fileSize+blockSize/2):], 0)
	assert.NoError(t, err, "can not read from file")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))
}

func TestStorage_ReadAt_OneFile(t *testing.T) {

	storage, files := prepareStorage("TestStorage_ReadAt_OneFile")

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	_, err := files[0].WriteAt(testData, fileSize/2)
	assert.NoError(t, err, "can not write to file")

	n, err := storage.ReadAt(data, fileSize/2)
	assert.NoError(t, err, "can not read from storage")
	assert.EqualValues(t, blockSize, n, "read bytes != block size")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))

	rand.Read(testData)

	_, err = files[1].WriteAt(testData, fileSize/2)
	assert.NoError(t, err, "can not write to file")

	n, err = storage.ReadAt(data, fileSize+fileSize/2)
	assert.NoError(t, err, "can not read from storage")
	assert.EqualValues(t, blockSize, n, "read bytes != block size")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))

}

func TestStorage_ReadAt_TwoFiles(t *testing.T) {

	storage, files := prepareStorage("TestStorage_ReadAt_TwoFiles")

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	// border write

	_, err := files[0].WriteAt(testData[:blockSize/2], fileSize-blockSize/2)
	assert.NoError(t, err, "can not write to file")

	_, err = files[1].WriteAt(testData[blockSize/2:], 0)
	assert.NoError(t, err, "can not write to file")

	n, err := storage.ReadAt(data, fileSize-blockSize/2)
	assert.NoError(t, err, "can not read from storage")
	assert.EqualValues(t, blockSize, n, "read bytes != block size")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))

}

func TestStorage_ReadAt_ThreeFiles(t *testing.T) {

	storage, files := prepareStorage("TestStorage_WriteAt_ThreeFiles")

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	// large block

	data = make([]byte, fileSize+blockSize)

	testData = make([]byte, fileSize+blockSize)
	rand.Read(testData)

	_, err := files[0].WriteAt(testData[:blockSize/2], fileSize-blockSize/2)
	assert.NoError(t, err, "can not write to file")

	_, err = files[1].WriteAt(testData[(blockSize/2):(fileSize+blockSize/2)], 0)
	assert.NoError(t, err, "can not write to file")

	_, err = files[2].WriteAt(testData[(fileSize+blockSize/2):], 0)
	assert.NoError(t, err, "can not write to file")

	n, err := storage.ReadAt(data, fileSize-blockSize/2)
	assert.NoError(t, err, "can not read from storage")
	assert.EqualValues(t, fileSize+blockSize, n, "read bytes != block size")

	assert.True(t, bytes.Compare(data, testData) == 0,
		fmt.Sprintf("bytes doesnt match \nwrote data: %v \nread data:  %v", testData, data))
}

func TestStorage_WriteAt_BreakBoundaries(t *testing.T) {

	storage, _ := prepareStorage("TestStorage_WriteAt_ThreeFiles")

	data := make([]byte, blockSize)
	_, err := storage.WriteAt(data, fileSize*3-blockSize/2)
	assert.Error(t, err, "write out of boundaries")

	_, err = storage.WriteAt(data, -blockSize)
	assert.Error(t, err, "write out of boundaries")
}

func TestStorage_ReadAt_BreakBoundaries(t *testing.T) {

	storage, _ := prepareStorage("TestStorage_WriteAt_ThreeFiles")

	data := make([]byte, blockSize)
	_, err := storage.ReadAt(data, fileSize*3-blockSize/2)
	assert.Error(t, err, "write out of boundaries")

	_, err = storage.ReadAt(data, -blockSize)
	assert.Error(t, err, "write out of boundaries")
}
