package torrent

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
)

const fileSize = 256 * 1024
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
		file, err := os.Open(path.Join(dir, filename))
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

	// first file write
	n, err := storage.WriteAt(testData, fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if n != blockSize {
		t.Error()
	}

	_, err = files[0].ReadAt(data, fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

	// second file write
	rand.Read(testData)

	n, err = storage.WriteAt(testData, fileSize+fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if n != blockSize {
		t.Error()
	}

	_, err = files[1].ReadAt(data, fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

}

func TestStorage_WriteAt_TwoFiles(t *testing.T) {

	storage, files := prepareStorage("TestStorage_WriteAt_TwoFiles")

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	// border write

	n, err := storage.WriteAt(testData, fileSize-blockSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if n != blockSize {
		t.Error()
	}

	_, err = files[0].ReadAt(data[:blockSize/2], fileSize-blockSize/2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = files[1].ReadAt(data[blockSize/2:], 0)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	if n != fileSize+blockSize {
		t.Error()
	}

	_, err = files[0].ReadAt(data[:blockSize/2], fileSize-blockSize/2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = files[1].ReadAt(data[(blockSize/2):(fileSize+blockSize/2)], 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = files[2].ReadAt(data[(fileSize+blockSize/2):], 0)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

}
