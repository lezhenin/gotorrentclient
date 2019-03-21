package torrent

import (
	"fmt"
	"log"
	"os"
	"path"
)

type Storage struct {
	files     []*os.File
	fileInfos []os.FileInfo
	totalSize int64
}

func NewStorage(info Info, basePath string) (s *Storage, err error) {

	s = new(Storage)

	for _, infoFile := range info.Files {
		filePath := path.Join(basePath, info.Name)

		for _, pathPart := range infoFile.Path {
			filePath = path.Join(filePath, pathPart)
		}

		// ignore all errors
		_ = os.Mkdir(path.Dir(filePath), 0777)

		file, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}

		err = file.Truncate(infoFile.Length)
		if err != nil {
			return nil, err
		}

		fileInfo, err := file.Stat()
		if err != nil {
			return nil, err
		}

		s.files = append(s.files, file)
		s.fileInfos = append(s.fileInfos, fileInfo)

		log.Printf("File \"%s\" is created: %d bytes",
			filePath, infoFile.Length)
	}

	s.totalSize = info.TotalLength

	return s, nil

}

func (s *Storage) ReadAt(b []byte, off int64) (n int, err error) {

	fileOffset, firstFileIndex, fileCount := s.convertToFileOffset(off, int64(len(b)))

	leftBytes := int64(len(b))
	readBytes := int64(0)

	currentOffset := int64(fileOffset)
	nextOffset := int64(0)

	blockSize := int64(0)

	for i := firstFileIndex; i < firstFileIndex+fileCount; i++ {

		fileInfo := s.fileInfos[i]

		if currentOffset+leftBytes < fileInfo.Size() {
			blockSize = leftBytes
		} else {
			blockSize = fileInfo.Size() - currentOffset
			nextOffset = 0
		}

		n, err := s.files[i].ReadAt(b[readBytes:readBytes+blockSize], currentOffset)

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

func (s *Storage) WriteAt(b []byte, off int64) (n int, err error) {

	fileOffset, firstFileIndex, fileCount := s.convertToFileOffset(off, int64(len(b)))

	leftBytes := int64(len(b))
	wroteBytes := int64(0)
	currentOffset := int64(fileOffset)
	nextOffset := int64(0)
	blockSize := int64(0)

	for i := firstFileIndex; i < firstFileIndex+fileCount; i++ {

		fileInfo := s.fileInfos[i]

		if currentOffset+leftBytes < fileInfo.Size() {
			blockSize = leftBytes
		} else {
			blockSize = fileInfo.Size() - currentOffset
			nextOffset = 0
		}

		n, err := s.files[i].WriteAt(b[wroteBytes:wroteBytes+blockSize], currentOffset)

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

func (s *Storage) convertToFileOffset(offset, length int64) (fileOffset int64, firstFileIndex, fileCount int) {

	fileOffset = offset
	fileCount = 1

	for index, fileInfo := range s.fileInfos {
		if fileOffset < fileInfo.Size() {
			firstFileIndex = index
			break
		}
		fileOffset -= fileInfo.Size()
	}

	fileSize := s.fileInfos[firstFileIndex].Size()

	for fileOffset+length > fileSize {
		fileCount += 1
		fileSize += s.fileInfos[firstFileIndex+fileCount-1].Size()
	}

	return fileOffset, firstFileIndex, fileCount

}