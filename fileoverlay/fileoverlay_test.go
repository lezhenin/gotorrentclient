package fileoverlay

import "testing"

var testFiles = []string{"/home/iurii/Documents/go/src/github.com/lezhenin/gotorrentclient/test/file1",
	"/home/iurii/Documents/go/src/github.com/lezhenin/gotorrentclient/test/file2"}

var fileLengths = []uint32{25 * 1024, 51 * 1024}

func TestFileOverlay_Write(t *testing.T) {

	//const testIndex = 29
	//
	//fo, err := NewFileOverlay(pieceSize, pieceCount, fileLengths, testFiles)
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//if fo.totalSize != fileLengths[0]+fileLengths[1] {
	//	t.Error("w")
	//}
	//
	//data := make([]byte, pieceSize-1*1024)
	//
	//for i := 0; i < len(data); i++ {
	//	data[i] = 0x25
	//}
	//
	//n, err := fo.Write(0, 512, data)
	//
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//if int(n) != len(data) {
	//	t.Error("w")
	//}
	//
	//n, data, err = fo.Read(0, 511, uint32(len(data)))
	//
	//if data[0] != 0 {
	//	t.Error("w")
	//}
	//
	//for i := 1; i < len(data); i++ {
	//	if data[i] != 0x25 {
	//		t.Error("w", i)
	//	}
	//}

}
