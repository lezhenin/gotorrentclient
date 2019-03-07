package fileoverlay

import (
	"fmt"
	"github.com/lezhenin/gotorrentclient/bitfield"
	"os"
	"path"
)

type FileInfo struct {
	file   *os.File
	path   string
	length uint32
}

type PieceInfo struct {
	length uint32
}

type FileOverlay struct {
	files       []FileInfo
	pieces      []PieceInfo
	totalLength uint32

	pieceBitField bitfield.Bitfield
}

func NewFileOverlay(pieceLength, pieceCount uint32, fileLengths []uint32, filePaths []string) (fo *FileOverlay, err error) {

	if len(fileLengths) != len(filePaths) {
		panic("new file overlay: file length slice and file path slice have different length")
	}

	fo = new(FileOverlay)
	fo.totalLength = uint32(0)

	// todo reopen instead creation
	for i := range filePaths {
		_ = os.Mkdir(path.Dir(filePaths[i]), 0755)

		file, err := os.Create(filePaths[i])
		if err != nil {
			return nil, err
		}

		err = file.Truncate(int64(fileLengths[i]))
		if err != nil {
			return nil, err
		}

		fo.files = append(fo.files, FileInfo{file, filePaths[i], fileLengths[i]})
		fo.totalLength += fileLengths[i]
	}

	for i := uint32(0); i < pieceCount; i++ {
		fo.pieces = append(fo.pieces, PieceInfo{pieceLength})
	}

	if fo.totalLength > pieceCount*pieceLength {
		panic("new file overlay: total length > piece count * piece length")
	}

	return fo, nil
}

func (fo *FileOverlay) Read(pieceIndex, start, length uint32) (n uint32, data []byte, err error) {

	data = make([]byte, length)

	offset, firstFileIndex, fileCount :=
		fo.convertToFileOffset(pieceIndex, start, length)

	leftBytes := length
	readBytes := uint32(0)
	currentOffset := int64(offset)
	nextOffset := int64(0)
	blockLength := uint32(0)

	for i := firstFileIndex; i < firstFileIndex+fileCount; i++ {

		fileInfo := fo.files[i]

		if leftBytes < fileInfo.length {
			blockLength = leftBytes
		} else {
			blockLength = fileInfo.length
			nextOffset = 0
		}

		n, err := fileInfo.file.ReadAt(data[readBytes:readBytes+blockLength], currentOffset)

		if err != nil {
			return 0, nil, err
		}

		if uint32(n) != blockLength {
			return 0, nil, fmt.Errorf("read: block length != n")
		}

		readBytes += blockLength
		leftBytes -= blockLength

		currentOffset = nextOffset

	}

	if leftBytes != 0 || readBytes != length {
		return 0, nil, fmt.Errorf("read: left bytes != 0 or read bytes != length")
	}

	return readBytes, data, err

}

func (fo *FileOverlay) Write(pieceIndex, start uint32, data []byte) (n uint32, err error) {

	offset, firstFileIndex, fileCount :=
		fo.convertToFileOffset(pieceIndex, start, uint32(len(data)))

	leftBytes := uint32(len(data))
	wroteBytes := uint32(0)
	currentOffset := int64(offset)
	nextOffset := int64(0)
	blockLength := uint32(0)

	for i := firstFileIndex; i < firstFileIndex+fileCount; i++ {

		fileInfo := fo.files[i]

		if leftBytes < fileInfo.length {
			blockLength = leftBytes
		} else {
			blockLength = fileInfo.length
			nextOffset = 0
		}

		n, err := fileInfo.file.WriteAt(data[wroteBytes:wroteBytes+blockLength], currentOffset)

		if err != nil {
			return 0, err
		}

		if uint32(n) != blockLength {
			return 0, fmt.Errorf("write: block length != n")
		}

		wroteBytes += blockLength
		leftBytes -= blockLength

		currentOffset = nextOffset

	}

	if leftBytes != 0 || wroteBytes != uint32(len(data)) {
		return 0, fmt.Errorf("write: left bytes != 0 or read bytes != length")
	}

	return wroteBytes, nil
}

func (fo *FileOverlay) convertToFileOffset(
	pieceIndex, start, length uint32) (
	offset uint32, firstFileIndex, fileCount int) {

	offset = start
	fileCount = 1

	for i := uint32(0); i < pieceIndex; i++ {
		offset += fo.pieces[i].length
	}

	for i := 0; i < len(fo.files); i++ {
		if offset < fo.files[i].length {
			firstFileIndex = i
			break
		}
		offset -= fo.files[i].length
	}

	fileCapacity := fo.files[firstFileIndex].length

	for offset+length > fileCapacity {
		fileCount += 1
		fileCapacity += fo.files[firstFileIndex+fileCount-1].length
	}
	return offset, firstFileIndex, fileCount

}
