package fileoverlay

import (
	"fmt"
	"os"
)

type FileOverlay struct {
	files     []*os.File
	fileInfos []os.FileInfo
	totalSize int64
}

func NewFileOverlay(files []*os.File) (fo *FileOverlay, err error) {

	fo = new(FileOverlay)

	fo.files = files
	fo.fileInfos = make([]os.FileInfo, len(files))

	fo.totalSize = int64(0)

	for index, file := range files {

		fileInfo, err := file.Stat()
		fo.fileInfos[index] = fileInfo

		if err != nil {
			return nil, err
		}

		fo.totalSize += fileInfo.Size()

		// todo check permissions
		//fi.Mode()

	}

	return fo, nil
}

func (fo *FileOverlay) ReadAt(b []byte, off int64) (n int, err error) {

	fileOffset, firstFileIndex, fileCount := fo.convertToFileOffset(off, int64(len(b)))

	leftBytes := int64(len(b))
	readBytes := int64(0)

	currentOffset := int64(fileOffset)
	nextOffset := int64(0)

	blockSize := int64(0)

	for i := firstFileIndex; i < firstFileIndex+fileCount; i++ {

		fileInfo := fo.fileInfos[i]

		if leftBytes < fileInfo.Size() {
			blockSize = leftBytes
		} else {
			blockSize = fileInfo.Size()
			nextOffset = 0
		}

		n, err := fo.files[i].ReadAt(b[readBytes:readBytes+blockSize], currentOffset)

		if err != nil {
			return 0, err
		}

		if int64(n) != blockSize {
			return 0, fmt.Errorf("read: block length != n")
		}

		readBytes += blockSize
		leftBytes -= blockSize

		currentOffset = nextOffset

	}

	if leftBytes != 0 || readBytes != int64(len(b)) {
		return 0, fmt.Errorf("read: left bytes != 0 or read bytes != length")
	}

	return int(readBytes), nil

}

func (fo *FileOverlay) WriteAt(b []byte, off int64) (n int, err error) {

	fileOffset, firstFileIndex, fileCount := fo.convertToFileOffset(off, int64(len(b)))

	leftBytes := int64(len(b))
	wroteBytes := int64(0)
	currentOffset := int64(fileOffset)
	nextOffset := int64(0)
	blockSize := int64(0)

	for i := firstFileIndex; i < firstFileIndex+fileCount; i++ {

		fileInfo := fo.fileInfos[i]

		if leftBytes < fileInfo.Size() {
			blockSize = leftBytes
		} else {
			blockSize = fileInfo.Size()
			nextOffset = 0
		}

		n, err := fo.files[i].WriteAt(b[wroteBytes:wroteBytes+blockSize], currentOffset)

		if err != nil {
			return 0, err
		}

		if int64(n) != blockSize {
			return 0, fmt.Errorf("write: block length != n")
		}

		wroteBytes += blockSize
		leftBytes -= blockSize

		currentOffset = nextOffset

	}

	if leftBytes != 0 || wroteBytes != int64(len(b)) {
		return 0, fmt.Errorf("write: left bytes != 0 or read bytes != length")
	}

	return int(wroteBytes), nil
}

func (fo *FileOverlay) convertToFileOffset(offset, length int64) (fileOffset int64, firstFileIndex, fileCount int) {

	fileOffset = offset
	fileCount = 1

	for index, fileInfo := range fo.fileInfos {
		if fileOffset < fileInfo.Size() {
			firstFileIndex = index
			break
		}
		fileOffset -= fileInfo.Size()
	}

	fileSize := fo.fileInfos[firstFileIndex].Size()

	for fileOffset+length > fileSize {
		fileCount += 1
		fileSize += fo.fileInfos[firstFileIndex+fileCount-1].Size()
	}
	return fileOffset, firstFileIndex, fileCount

}
