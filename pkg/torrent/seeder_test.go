package torrent

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

func makeTestSeeder(infoHash, myPeerId []byte) (seeder *Seeder, messages chan Message) {

	messages = make(chan Message, 1)

	seeder, err := NewSeeder(infoHash, myPeerId, messages)

	if err != nil {
		panic(err)
	}

	return seeder, messages
}

func TestSeeder_Dial(t *testing.T) {

	infoHash := make([]byte, 20)
	myPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(myPeerId)

	seeder, _ := makeTestSeeder(infoHash, myPeerId)

	interiorConn, exteriorConn := net.Pipe()

	var wait sync.WaitGroup

	wait.Add(1)

	go func() {
		defer wait.Done()
		err := seeder.Dial(interiorConn)
		assert.NoError(t, err, "seeder dial finished with error")
	}()

	stringLength := make([]byte, 1)
	protocolString := make([]byte, 19)
	peerId := make([]byte, 20)
	extensions := make([]byte, 8)

	_, err := exteriorConn.Read(stringLength)
	assert.NoError(t, err, "can not read from connection")
	assert.EqualValues(t, 19, stringLength[0], "string length is wrong")

	_, err = exteriorConn.Read(protocolString)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(protocolString, []byte("BitTorrent protocol")) == 0,
		"protocol string is wrong")

	_, err = exteriorConn.Read(extensions)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(extensions, []byte{0, 0, 0, 0, 0, 0, 0, 0}) == 0,
		"unexpected extension bits")

	_, err = exteriorConn.Read(infoHash)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(infoHash, seeder.InfoHash) == 0,
		"info hash is wrong")

	_, err = exteriorConn.Read(peerId)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(peerId, seeder.MyPeerId) == 0,
		"my peer id is wrong")

	rand.Read(stringLength)

	protocolString = make([]byte, stringLength[0])

	rand.Read(protocolString)
	rand.Read(peerId)
	rand.Read(extensions)

	_, err = exteriorConn.Write(stringLength)
	assert.NoError(t, err, "can not read from connection")

	_, err = exteriorConn.Write(protocolString)
	assert.NoError(t, err, "can not read from connection")

	_, err = exteriorConn.Write(extensions)
	assert.NoError(t, err, "can not read from connection")

	_, err = exteriorConn.Write(infoHash)
	assert.NoError(t, err, "can not read from connection")

	_, err = exteriorConn.Write(peerId)
	assert.NoError(t, err, "can not read from connection")

	wait.Wait()

	assert.True(t, bytes.Compare(seeder.PeerId, peerId) == 0,
		"peer id is wrong")
}

func TestSeeder_Dial_Timout(t *testing.T) {

	if testing.Short() {
		t.Skip("skip in short mode: test duration is about 15s")
	}

	infoHash := make([]byte, 20)
	myPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(myPeerId)

	seeder, _ := makeTestSeeder(infoHash, myPeerId)

	interiorConn, exteriorConn := net.Pipe()

	var wait sync.WaitGroup

	wait.Add(1)

	go func() {
		defer wait.Done()
		err := seeder.Dial(interiorConn)
		assert.Error(t, err, "seeder dial finished without error")
	}()

	stringLength := make([]byte, 1)
	protocolString := make([]byte, 19)
	peerId := make([]byte, 20)
	extensions := make([]byte, 8)

	_, err := exteriorConn.Read(stringLength)
	assert.NoError(t, err, "can not read from connection")
	assert.EqualValues(t, 19, stringLength[0], "string length is wrong")

	_, err = exteriorConn.Read(protocolString)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(protocolString, []byte("BitTorrent protocol")) == 0,
		"protocol string is wrong")

	_, err = exteriorConn.Read(extensions)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(extensions, []byte{0, 0, 0, 0, 0, 0, 0, 0}) == 0,
		"unexpected extension bits")

	_, err = exteriorConn.Read(infoHash)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(infoHash, seeder.InfoHash) == 0,
		"info hash is wrong")

	_, err = exteriorConn.Read(peerId)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(peerId, seeder.MyPeerId) == 0,
		"my peer id is wrong")

	time.Sleep(15 * time.Second)

	wait.Wait()

	seeder.Close()

}

func TestSeeder_Accept(t *testing.T) {

	infoHash := make([]byte, 20)
	myPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(myPeerId)

	seeder, _ := makeTestSeeder(infoHash, myPeerId)

	interiorConn, exteriorConn := net.Pipe()

	var wait sync.WaitGroup

	wait.Add(1)

	go func() {
		defer wait.Done()
		err := seeder.Accept(interiorConn)
		assert.NoError(t, err, "seeder accept finished with error")
	}()

	stringLength := make([]byte, 1)
	peerId := make([]byte, 20)
	extensions := make([]byte, 8)

	rand.Read(stringLength)

	protocolString := make([]byte, stringLength[0])

	rand.Read(protocolString)
	rand.Read(peerId)
	rand.Read(extensions)

	_, err := exteriorConn.Write(stringLength)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Write(protocolString)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Write(extensions)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Write(infoHash)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Write(peerId)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Read(stringLength)
	assert.NoError(t, err, "can not read from connection")

	assert.EqualValues(t, 19, stringLength[0], "string length is wrong")

	protocolString = make([]byte, 19)

	_, err = exteriorConn.Read(protocolString)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(protocolString, []byte("BitTorrent protocol")) == 0,
		"protocol string is wrong")

	_, err = exteriorConn.Read(extensions)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(extensions, []byte{0, 0, 0, 0, 0, 0, 0, 0}) == 0,
		"unexpected extension bits")

	_, err = exteriorConn.Read(infoHash)
	assert.NoError(t, err, "can not read from connection")

	if bytes.Compare(infoHash, seeder.InfoHash) != 0 {
		t.Errorf("Info hash is wrong: %v != %v", infoHash, seeder.InfoHash)
	}

	_, err = exteriorConn.Read(myPeerId)
	assert.NoError(t, err, "can not read from connection")
	assert.True(t, bytes.Compare(infoHash, seeder.InfoHash) == 0,
		"info hash is wrong")

	wait.Wait()

	assert.True(t, bytes.Compare(seeder.PeerId, peerId) == 0,
		"peer id is wrong")

	seeder.Close()

}

func TestSeeder_Accept_WrongHash(t *testing.T) {

	infoHash := make([]byte, 20)
	myPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(myPeerId)

	seeder, _ := makeTestSeeder(infoHash, myPeerId)

	interiorConn, exteriorConn := net.Pipe()

	var wait sync.WaitGroup

	wait.Add(1)

	go func() {
		defer wait.Done()
		err := seeder.Accept(interiorConn)
		assert.Error(t, err, "seeder dial finished without error")
	}()

	stringLength := make([]byte, 1)
	peerId := make([]byte, 20)
	extensions := make([]byte, 8)

	rand.Read(stringLength)

	protocolString := make([]byte, stringLength[0])

	rand.Read(protocolString)
	rand.Read(peerId)
	rand.Read(extensions)

	wrongInfoHash := make([]byte, 20)

	for i := range wrongInfoHash {
		wrongInfoHash[i] = infoHash[i] + 1 + peerId[i]/2
	}

	_, err := exteriorConn.Write(stringLength)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Write(protocolString)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Write(extensions)
	assert.NoError(t, err, "can not write to connection")

	_, err = exteriorConn.Write(wrongInfoHash)
	assert.NoError(t, err, "can not write to connection")

	wait.Wait()

	seeder.Close()
}

func TestSeeder_Start_PieceTimout(t *testing.T) {

	if testing.Short() {
		t.Skip("skip in short mode: test duration is about 15s")
	}

	infoHash := make([]byte, 20)
	firstPeerId := make([]byte, 20)
	secondPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(firstPeerId)
	rand.Read(secondPeerId)

	firstSeeder, _ := makeTestSeeder(infoHash, firstPeerId)
	secondSeeder, _ := makeTestSeeder(infoHash, secondPeerId)

	firstConn, secondConn := net.Pipe()

	var wait sync.WaitGroup

	wait.Add(2)

	go func() {
		defer wait.Done()
		err := firstSeeder.Dial(firstConn)
		assert.NoError(t, err, "seeder dial finished with error")
	}()

	go func() {
		defer wait.Done()
		err := secondSeeder.Accept(secondConn)
		assert.NoError(t, err, "seeder accept finished with error")

	}()

	wait.Wait()

	var mutex sync.Mutex
	closed := false

	wait.Add(2)

	go func() {
		defer wait.Done()
		firstSeeder.Start()
		mutex.Lock()
		defer mutex.Unlock()
		closed = true
	}()

	go func() {
		defer wait.Done()
		secondSeeder.Start()
	}()

	payload := make([]byte, 215)
	rand.Read(payload)

	firstSeeder.outcoming <- Message{Request, payload, nil}

	time.Sleep(16 * time.Second)

	mutex.Lock()
	assert.True(t, closed, "seeder is running ")
	mutex.Unlock()

	firstSeeder.Close()
	secondSeeder.Close()

	wait.Wait()

}

func TestSeeder_Start(t *testing.T) {

	infoHash := make([]byte, 20)
	firstPeerId := make([]byte, 20)
	secondPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(firstPeerId)
	rand.Read(secondPeerId)

	firstSeeder, _ := makeTestSeeder(infoHash, firstPeerId)
	secondSeeder, _ := makeTestSeeder(infoHash, secondPeerId)

	firstConn, secondConn := net.Pipe()

	var wait sync.WaitGroup

	wait.Add(2)

	go func() {
		defer wait.Done()
		err := firstSeeder.Dial(firstConn)
		assert.NoError(t, err, "seeder dial finished with error")

	}()

	go func() {
		defer wait.Done()
		err := secondSeeder.Accept(secondConn)
		assert.NoError(t, err, "seeder accept finished with error")

	}()

	wait.Wait()

	wait.Add(2)

	go func() {
		defer wait.Done()
		firstSeeder.Start()
	}()

	go func() {
		defer wait.Done()
		secondSeeder.Start()
	}()

	firstSeeder.outcoming <- Message{KeepAlive, nil, nil}
	message := <-secondSeeder.incoming
	assert.EqualValues(t, KeepAlive, message.Id, "wrong message id")
	assert.Nil(t, message.Payload, "wrong message payload")

	firstSeeder.outcoming <- Message{Interested, nil, nil}
	message = <-secondSeeder.incoming
	assert.EqualValues(t, Interested, message.Id, "wrong message id")
	assert.Nil(t, message.Payload, "wrong message payload")

	payload := make([]byte, 215)
	rand.Read(payload)

	firstSeeder.outcoming <- Message{Bitfield, payload, nil}
	message = <-secondSeeder.incoming
	assert.EqualValues(t, Bitfield, message.Id, "wrong message id")
	assert.True(t, bytes.Compare(message.Payload, payload) == 0,
		"wrong message payload")

	firstSeeder.Close()
	secondSeeder.Close()

	wait.Wait()

}

func TestSeeder_Start_Disconnect(t *testing.T) {

	infoHash := make([]byte, 20)
	firstPeerId := make([]byte, 20)
	secondPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(firstPeerId)
	rand.Read(secondPeerId)

	firstSeeder, _ := makeTestSeeder(infoHash, firstPeerId)
	secondSeeder, _ := makeTestSeeder(infoHash, secondPeerId)

	firstConn, secondConn := net.Pipe()

	var wait sync.WaitGroup

	wait.Add(2)

	go func() {
		defer wait.Done()
		err := firstSeeder.Dial(firstConn)
		assert.NoError(t, err, "seeder dial finished with error")

	}()

	go func() {
		defer wait.Done()
		err := secondSeeder.Accept(secondConn)
		assert.NoError(t, err, "seeder accept finished with error")

	}()

	wait.Wait()

	wait.Add(2)

	go func() {
		defer wait.Done()
		firstSeeder.Start()
	}()

	go func() {
		defer wait.Done()
		secondSeeder.Start()
	}()

	firstSeeder.outcoming <- Message{KeepAlive, nil, nil}
	message := <-secondSeeder.incoming
	assert.EqualValues(t, KeepAlive, message.Id, "wrong message id")
	assert.Nil(t, message.Payload, "wrong message payload")

	_ = firstConn.Close()

	wait.Wait()

	firstSeeder.Close()
	secondSeeder.Close()

}

func TestSeeder_MakePayload(t *testing.T) {

	pieceIndex := rand.Uint32()
	offset := rand.Uint32()
	length := rand.Uint32()

	payload := MakeHavePayload(pieceIndex)
	parsedPieceIndex := binary.BigEndian.Uint32(payload)
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "have payload is wrong")

	payload = MakeRequestPayload(pieceIndex, offset, length)
	parsedPieceIndex = binary.BigEndian.Uint32(payload[0:4])
	parsedOffset := binary.BigEndian.Uint32(payload[4:8])
	parsedLength := binary.BigEndian.Uint32(payload[8:12])
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "request payload is wrong")
	assert.EqualValues(t, offset, parsedOffset, "request payload is wrong")
	assert.EqualValues(t, length, parsedLength, "request payload is wrong")

	payload = MakeCancelPayload(pieceIndex, offset, length)
	parsedPieceIndex = binary.BigEndian.Uint32(payload[0:4])
	parsedOffset = binary.BigEndian.Uint32(payload[4:8])
	parsedLength = binary.BigEndian.Uint32(payload[8:12])
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "cancel payload is wrong")
	assert.EqualValues(t, offset, parsedOffset, "cancel payload is wrong")
	assert.EqualValues(t, length, parsedLength, "cancel payload is wrong")

	block := make([]byte, 256)
	rand.Read(block)

	payload = MakePiecePayload(pieceIndex, offset, block)
	parsedPieceIndex = binary.BigEndian.Uint32(payload[0:4])
	parsedOffset = binary.BigEndian.Uint32(payload[4:8])
	parsedBlock := payload[8:]
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "piece payload is wrong")
	assert.EqualValues(t, offset, parsedOffset, "piece payload is wrong")
	assert.True(t, bytes.Compare(parsedBlock, block) == 0, "piece payload is wrong")

}

func TestSeeder_MakeAndParsePayload(t *testing.T) {

	pieceIndex := rand.Uint32()
	offset := rand.Uint32()
	length := rand.Uint32()

	payload := MakeHavePayload(pieceIndex)
	parsedPieceIndex, err := ParseHavePayload(payload)
	assert.NoError(t, err, "can not parse have payload")
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "make and parse dont correspond")

	_, err = ParseHavePayload(payload[1:])
	assert.Error(t, err, "parse have payload with wrong length")

	payload = MakeRequestPayload(pieceIndex, offset, length)
	parsedPieceIndex, parsedOffset, parsedLength, err := ParseRequestPayload(payload)
	assert.NoError(t, err, "can not parse request payload")
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "make and parse dont correspond")
	assert.EqualValues(t, offset, parsedOffset, "make and parse dont correspond")
	assert.EqualValues(t, length, parsedLength, "make and parse dont correspond")

	_, _, _, err = ParseRequestPayload(payload[1:])
	assert.Error(t, err, "parse request payload with wrong length")

	payload = MakeCancelPayload(pieceIndex, offset, length)
	parsedPieceIndex, parsedOffset, parsedLength, err = ParseCancelPayload(payload)
	assert.NoError(t, err, "can not parse cancel payload")
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "make and parse dont correspond")
	assert.EqualValues(t, offset, parsedOffset, "make and parse dont correspond")
	assert.EqualValues(t, length, parsedLength, "make and parse dont correspond")

	_, _, _, err = ParseCancelPayload(payload[1:])
	assert.Error(t, err, "parse cancel payload with wrong length")

	block := make([]byte, 256)
	rand.Read(block)

	payload = MakePiecePayload(pieceIndex, offset, block)
	parsedPieceIndex, parsedOffset, parsedBlock, err := ParsePiecePayload(payload)
	assert.NoError(t, err, "can not parse piece payload")
	assert.EqualValues(t, pieceIndex, parsedPieceIndex, "make and parse dont correspond")
	assert.EqualValues(t, offset, parsedOffset, "make and parse dont correspond")
	assert.True(t, bytes.Compare(parsedBlock, block) == 0, "make and parse dont correspond")

	_, _, _, err = ParsePiecePayload(payload[:7])
	assert.Error(t, err, "parse piece payload with wrong length")

}
