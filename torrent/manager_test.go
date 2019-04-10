package torrent

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"net"
	"sync"
	"testing"
)

func TestManager_Start(t *testing.T) {

	filename := "../test/DA3F.torrent"
	metadata, err := NewMetadata(filename)
	assert.NoError(t, err, "can not decode metadata")

	bitfieldLength := uint(metadata.Info.PieceCount) / 8
	if metadata.Info.PieceCount%8 > 0 {
		bitfieldLength += 1
	}

	state := NewState(uint64(metadata.Info.TotalLength), bitfieldLength)

	tempDir, err := ioutil.TempDir("", "TestManager_Start")
	assert.NoError(t, err, "can not temp dir")

	storage, err := NewStorage(metadata.Info, tempDir)
	assert.NoError(t, err, "can not create storage")

	peerId := make([]byte, 20)
	rand.Read(peerId)

	infoHash := metadata.Info.HashSHA1

	manager := NewManager(peerId, infoHash, &metadata.Info, state, storage)

	var wait sync.WaitGroup
	wait.Add(1)

	go func() {
		defer wait.Done()
		manager.Start()
	}()

	interiorConn, exteriorConn := net.Pipe()

	exteriorPeerId := make([]byte, 20)
	rand.Read(exteriorPeerId)

	exteriorSeeder, _ := makeTestSeeder(infoHash, exteriorPeerId)

	exteriorStorage, err := NewStorage(metadata.Info, "../test")

	fmt.Println(exteriorPeerId)

	var seederWait sync.WaitGroup
	seederWait.Add(1)

	go func() {
		defer seederWait.Done()
		err := exteriorSeeder.Accept(exteriorConn)
		assert.NoError(t, err, "can not accept seeder connection")
	}()

	err = manager.AddSeeder(interiorConn, false)
	assert.NoError(t, err, "manager can not accept seeder connection")

	seederWait.Wait()

	seederWait.Add(1)
	go func() {
		defer seederWait.Done()
		exteriorSeeder.Start()
	}()

	exteriorSeeder.outcoming <- Message{Have, MakeHavePayload(3), nil}
	receivedMessage := <-exteriorSeeder.incoming
	assert.EqualValues(t, Interested, receivedMessage.Id, "unexpected received message")

	exteriorSeeder.outcoming <- Message{Unchoke, nil, nil}
	receivedMessage = <-exteriorSeeder.incoming
	assert.EqualValues(t, Request, receivedMessage.Id, "unexpected received message")

	index, begin, length, err := ParseRequestPayload(receivedMessage.Payload)
	assert.NoError(t, err, "can not parse request payload")
	assert.EqualValues(t, 3, index, "request wrong index")

	data := make([]byte, length)
	offset := int64(index*uint32(metadata.Info.PieceLength) + begin)
	_, err = exteriorStorage.ReadAt(data, offset)
	assert.NoError(t, err, "can not read from storage")

	exteriorSeeder.outcoming <- Message{Piece, MakePiecePayload(index, begin, data), nil}
	receivedMessage = <-exteriorSeeder.incoming
	assert.EqualValues(t, Request, receivedMessage.Id, "unexpected received message")

	index, begin, length, err = ParseRequestPayload(receivedMessage.Payload)
	assert.NoError(t, err, "can not parse request payload")
	assert.EqualValues(t, 3, index, "request wrong index")

	data = make([]byte, length)
	offset = int64(index*uint32(metadata.Info.PieceLength) + begin)
	_, err = exteriorStorage.ReadAt(data, offset)
	assert.NoError(t, err, "can not read from storage")

	exteriorSeeder.outcoming <- Message{Piece, MakePiecePayload(index, begin, data), nil}
	receivedMessage = <-exteriorSeeder.incoming
	assert.EqualValues(t, NotInterested, receivedMessage.Id, "unexpected received message")

	manager.Stop()
	exteriorSeeder.Close()
	seederWait.Wait()
	wait.Wait()

}
