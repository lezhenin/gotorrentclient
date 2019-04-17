package fileoverlay

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
)

const fileSize = 256 * 1024
const blockSize = 16 * 1024

func TestFileOverlay_WriteAt(t *testing.T) {

	var files []*os.File

	for i := 0; i < 3; i++ {

		file, err := ioutil.TempFile("", "TestFileOverlay_Write")
		if err != nil {
			t.Fatal(err)
		}

		if err = file.Truncate(fileSize); err != nil {
			t.Fatal(err)
		}

		files = append(files, file)
	}

	overlay, err := NewFileOverlay(files)
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	// first file write
	n, err := overlay.WriteAt(testData, fileSize/2)
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

	n, err = overlay.WriteAt(testData, fileSize+fileSize/2)
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

	// border write

	rand.Read(testData)

	n, err = overlay.WriteAt(testData, fileSize-blockSize/2)
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

	// large block

	data = make([]byte, fileSize+blockSize)

	testData = make([]byte, fileSize+blockSize)
	rand.Read(testData)

	n, err = overlay.WriteAt(testData, fileSize-blockSize/2)
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

func TestFileOverlay_ReadAt(t *testing.T) {

	var files []*os.File

	for i := 0; i < 3; i++ {

		file, err := ioutil.TempFile("", "TestFileOverlay_Write")
		if err != nil {
			t.Fatal(err)
		}

		if err = file.Truncate(fileSize); err != nil {
			t.Fatal(err)
		}

		files = append(files, file)
	}

	overlay, err := NewFileOverlay(files)
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, blockSize)

	testData := make([]byte, blockSize)
	rand.Read(testData)

	// first file write

	_, err = files[0].WriteAt(testData, fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	n, err := overlay.ReadAt(data, fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if n != blockSize {
		t.Error()
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

	// second file write

	rand.Read(testData)

	_, err = files[1].WriteAt(testData, fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	n, err = overlay.ReadAt(data, fileSize+fileSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if n != blockSize {
		t.Error()
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

	// border write

	rand.Read(testData)

	_, err = files[0].WriteAt(testData[:blockSize/2], fileSize-blockSize/2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = files[1].WriteAt(testData[blockSize/2:], 0)
	if err != nil {
		t.Fatal(err)
	}

	n, err = overlay.ReadAt(data, fileSize-blockSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if n != blockSize {
		t.Error()
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

	// large block

	data = make([]byte, fileSize+blockSize)

	testData = make([]byte, fileSize+blockSize)
	rand.Read(testData)

	_, err = files[0].WriteAt(testData[:blockSize/2], fileSize-blockSize/2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = files[1].WriteAt(testData[(blockSize/2):(fileSize+blockSize/2)], 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = files[2].WriteAt(testData[(fileSize+blockSize/2):], 0)
	if err != nil {
		t.Fatal(err)
	}

	n, err = overlay.ReadAt(data, fileSize-blockSize/2)
	if err != nil {
		t.Fatal(err)
	}

	if n != fileSize+blockSize {
		t.Error()
	}

	if bytes.Compare(data, testData) != 0 {
		t.Errorf("\nWrote data: %v \nRead data:  %v", testData, data)
	}

}
