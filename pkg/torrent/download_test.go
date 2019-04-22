package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"net"
	"sync"
	"testing"
)

func TestDownload_Start(t *testing.T) {

	conn, err := net.ListenPacket("udp", ":8000")
	assert.NoError(t, err, "can not create listener")

	tempDir, err := ioutil.TempDir("", "TestDownload_Start")

	metadata, err := NewMetadata("../../test/test_download/test_data_localhost.torrent")
	assert.NoError(t, err, "can not read metadata")

	download, err := NewDownload(metadata, tempDir)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		err = download.Start()
		assert.NoError(t, err, "download finished with error")
	}()

	go func() {

		defer wg.Done()

		assert.NoError(t, err, "can not accept upd connection")

		buffer := make([]byte, 16)

		n, addr, err := conn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

		trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
		actionId := binary.BigEndian.Uint32(buffer[8:12])
		assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
		assert.EqualValues(t, 0, actionId, "wrong action ID")

		copy(buffer[0:8], buffer[8:16])
		rand.Read(buffer[8:16])
		connectionId := binary.BigEndian.Uint64(buffer[8:16])

		n, err = conn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

		buffer = make([]byte, 98)
		n, addr, err = conn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

		transactionId := binary.BigEndian.Uint32(buffer[12:16])

		assert.EqualValues(t, connectionId, binary.BigEndian.Uint64(buffer[0:8]), "wrong connection id")
		assert.EqualValues(t, 1, binary.BigEndian.Uint32(buffer[8:12]), "wrong action id")
		assert.True(t, bytes.Compare(metadata.Info.HashSHA1, buffer[16:36]) == 0, "wrong info hash")
		assert.EqualValues(t, 0, binary.BigEndian.Uint64(buffer[56:64]), "wrong downloaded")
		assert.EqualValues(t, metadata.Info.TotalLength, binary.BigEndian.Uint64(buffer[64:72]), "wrong left")
		assert.EqualValues(t, 0, binary.BigEndian.Uint64(buffer[72:80]), "wrong uploaded")
		assert.EqualValues(t, 2, binary.BigEndian.Uint32(buffer[80:84]), "wrong event id")
		assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[84:88]), "wrong ip address")
		assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[88:92]), "wrong key")
		assert.EqualValues(t, 100, binary.BigEndian.Uint32(buffer[92:96]), "wrong peer count")
		assert.EqualValues(t, 8861, binary.BigEndian.Uint16(buffer[96:98]), "wrong port")

		buffer = make([]byte, 20)
		binary.BigEndian.PutUint32(buffer[0:4], 1)
		binary.BigEndian.PutUint32(buffer[4:8], transactionId)
		binary.BigEndian.PutUint32(buffer[8:12], 1)
		binary.BigEndian.PutUint32(buffer[12:16], 0)
		binary.BigEndian.PutUint32(buffer[16:20], 0)

		n, err = conn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

		buffer = make([]byte, 98)
		n, addr, err = conn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

		transactionId = binary.BigEndian.Uint32(buffer[12:16])

		buffer = make([]byte, 20)
		binary.BigEndian.PutUint32(buffer[0:4], 1)
		binary.BigEndian.PutUint32(buffer[4:8], transactionId)
		binary.BigEndian.PutUint32(buffer[8:12], 100)
		binary.BigEndian.PutUint32(buffer[12:16], 0)
		binary.BigEndian.PutUint32(buffer[16:20], 0)

		n, err = conn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

		go func() {
			defer wg.Done()
			download.Stop()
		}()

		buffer = make([]byte, 98)
		n, addr, err = conn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

		transactionId = binary.BigEndian.Uint32(buffer[12:16])

		buffer = make([]byte, 20)
		binary.BigEndian.PutUint32(buffer[0:4], 1)
		binary.BigEndian.PutUint32(buffer[4:8], transactionId)
		binary.BigEndian.PutUint32(buffer[8:12], 100)
		binary.BigEndian.PutUint32(buffer[12:16], 0)
		binary.BigEndian.PutUint32(buffer[16:20], 0)

		n, err = conn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

	}()

	wg.Wait()

	conn.Close()

}

func TestDownload_Finish(t *testing.T) {

	seederListener, err := net.Listen("tcp", ":8999")
	assert.NoError(t, err, "can not create listener")

	trackerConn, err := net.ListenPacket("udp", ":8000")
	assert.NoError(t, err, "can not create listener")

	tempDir, err := ioutil.TempDir("", "TestDownload_Start")

	metadata, err := NewMetadata("../../test/test_download/test_data_localhost.torrent")
	assert.NoError(t, err, "can not read metadata")

	storage, err := NewStorage(metadata.Info, "../../test/test_download/")

	download, err := NewDownload(metadata, tempDir)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		err = download.Start()
		assert.NoError(t, err, "download finished with error")
	}()

	go func() {

		defer wg.Done()

		assert.NoError(t, err, "can not accept upd connection")

		buffer := make([]byte, 16)

		n, addr, err := trackerConn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

		trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
		actionId := binary.BigEndian.Uint32(buffer[8:12])
		assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
		assert.EqualValues(t, 0, actionId, "wrong action ID")

		copy(buffer[0:8], buffer[8:16])
		rand.Read(buffer[8:16])
		connectionId := binary.BigEndian.Uint64(buffer[8:16])

		n, err = trackerConn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

		buffer = make([]byte, 98)
		n, addr, err = trackerConn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

		transactionId := binary.BigEndian.Uint32(buffer[12:16])

		assert.EqualValues(t, connectionId, binary.BigEndian.Uint64(buffer[0:8]), "wrong connection id")
		assert.EqualValues(t, 1, binary.BigEndian.Uint32(buffer[8:12]), "wrong action id")
		assert.True(t, bytes.Compare(metadata.Info.HashSHA1, buffer[16:36]) == 0, "wrong info hash")
		assert.EqualValues(t, 0, binary.BigEndian.Uint64(buffer[56:64]), "wrong downloaded")
		assert.EqualValues(t, metadata.Info.TotalLength, binary.BigEndian.Uint64(buffer[64:72]), "wrong left")
		assert.EqualValues(t, 0, binary.BigEndian.Uint64(buffer[72:80]), "wrong uploaded")
		assert.EqualValues(t, 2, binary.BigEndian.Uint32(buffer[80:84]), "wrong event id")
		assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[84:88]), "wrong ip address")
		assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[88:92]), "wrong key")
		assert.EqualValues(t, 100, binary.BigEndian.Uint32(buffer[92:96]), "wrong peer count")
		assert.EqualValues(t, 8861, binary.BigEndian.Uint16(buffer[96:98]), "wrong port")

		buffer = make([]byte, 26)
		binary.BigEndian.PutUint32(buffer[0:4], 1)
		binary.BigEndian.PutUint32(buffer[4:8], transactionId)
		binary.BigEndian.PutUint32(buffer[8:12], 100)
		binary.BigEndian.PutUint32(buffer[12:16], 0)
		binary.BigEndian.PutUint32(buffer[16:20], 1)
		binary.BigEndian.PutUint16(buffer[24:26], 8999)

		buffer[20] = 127
		buffer[21] = 0
		buffer[22] = 0
		buffer[23] = 1

		n, err = trackerConn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

		buffer = make([]byte, 98)
		n, addr, err = trackerConn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

		assert.EqualValues(t, 1, binary.BigEndian.Uint32(buffer[80:84]), "wrong event id")

		transactionId = binary.BigEndian.Uint32(buffer[12:16])

		buffer = make([]byte, 20)
		binary.BigEndian.PutUint32(buffer[0:4], 1)
		binary.BigEndian.PutUint32(buffer[4:8], transactionId)
		binary.BigEndian.PutUint32(buffer[8:12], 100)
		binary.BigEndian.PutUint32(buffer[12:16], 0)
		binary.BigEndian.PutUint32(buffer[16:20], 0)

		n, err = trackerConn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

		buffer = make([]byte, 98)
		n, addr, err = trackerConn.ReadFrom(buffer)
		assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

		transactionId = binary.BigEndian.Uint32(buffer[12:16])

		buffer = make([]byte, 20)
		binary.BigEndian.PutUint32(buffer[0:4], 1)
		binary.BigEndian.PutUint32(buffer[4:8], transactionId)
		binary.BigEndian.PutUint32(buffer[8:12], 100)
		binary.BigEndian.PutUint32(buffer[12:16], 0)
		binary.BigEndian.PutUint32(buffer[16:20], 0)

		n, err = trackerConn.WriteTo(buffer, addr)
		assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

	}()

	go func() {

		defer wg.Done()

		peerId := make([]byte, 20)
		rand.Read(peerId)

		fmt.Println(peerId)

		inc := make(chan Message)
		seeder, err := NewSeeder(metadata.Info.HashSHA1, peerId, inc)
		assert.NoError(t, err, "can not create seeder")

		seederConn, err := seederListener.Accept()
		assert.NoError(t, err, "can not accept tcp connection")

		err = seeder.Accept(seederConn)
		assert.NoError(t, err, "can not accept seeder connection")

		var seederWg sync.WaitGroup
		seederWg.Add(1)

		go func() {
			defer seederWg.Done()
			seeder.Start()
		}()

		bitfieldLength := int(metadata.Info.PieceCount) / 8
		if metadata.Info.PieceCount%8 > 0 {
			bitfieldLength += 1
		}

		bitfieldBytes := make([]byte, bitfieldLength)
		for i := 0; i < bitfieldLength; i++ {
			bitfieldBytes[i] = 0xFF
		}

		fmt.Println(download.State.BitfieldBytes())
		fmt.Println(bitfieldBytes)

		seeder.outcoming <- Message{Bitfield, bitfieldBytes, nil}

		receivedMessage := <-seeder.incoming
		assert.EqualValues(t, Interested, receivedMessage.Id, "unexpected received message")

		seeder.outcoming <- Message{Unchoke, nil, nil}

		blockCount := metadata.Info.TotalLength / blockSize
		if metadata.Info.TotalLength%blockSize > 0 {
			blockCount += 1
		}

		for i := 0; i < int(blockCount); i++ {

			receivedMessage = <-seeder.incoming
			assert.EqualValues(t, Request, receivedMessage.Id, "unexpected received message")
			requestPayload := receivedMessage.Payload

			index, begin, length, err := ParseRequestPayload(requestPayload)
			data := make([]byte, length)
			offset := int64(index*uint32(metadata.Info.PieceLength) + begin)
			_, err = storage.ReadAt(data, offset)
			assert.NoError(t, err, "can not read from storage")

			piecePayload := MakePiecePayload(index, begin, data)
			seeder.outcoming <- Message{Piece, piecePayload, nil}
		}

		receivedMessage = <-seeder.incoming
		assert.EqualValues(t, NotInterested, receivedMessage.Id, "unexpected received message")

		seeder.Close()

		seederWg.Wait()

	}()

	<-download.Done
	assert.True(t, download.State.Finished(), "wrong download state")
	download.Stop()

	wg.Wait()

	trackerConn.Close()
	seederListener.Close()
}
