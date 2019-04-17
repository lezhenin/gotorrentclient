package torrent

import (
	"bytes"
	"encoding/binary"
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
		if err != nil {
			t.Error(err)
		}
	}()

	stringLength := make([]byte, 1)
	protocolString := make([]byte, 19)
	peerId := make([]byte, 20)
	extensions := make([]byte, 8)

	if _, err := exteriorConn.Read(stringLength); err != nil {
		panic(err)
	}

	if stringLength[0] != 19 {
		t.Errorf("String length is wrong: %d", stringLength[0])
	}

	if _, err := exteriorConn.Read(protocolString); err != nil {
		panic(err)
	}

	if bytes.Compare(protocolString, []byte("BitTorrent protocol")) != 0 {
		t.Errorf("Protocol tring is wrong: %s", string(protocolString))
	}

	if _, err := exteriorConn.Read(extensions); err != nil {
		panic(err)
	}

	if bytes.Compare(extensions, []byte{0, 0, 0, 0, 0, 0, 0, 0}) != 0 {
		t.Errorf("Extension bits is not null: %v", extensions)
	}

	if _, err := exteriorConn.Read(infoHash); err != nil {
		panic(err)
	}

	if bytes.Compare(infoHash, seeder.InfoHash) != 0 {
		t.Errorf("Info hash is wrong: %v != %v", infoHash, seeder.InfoHash)
	}

	if _, err := exteriorConn.Read(peerId); err != nil {
		panic(err)
	}

	if bytes.Compare(peerId, seeder.MyPeerId) != 0 {
		t.Errorf("My peer id is wrong: %v != %v", peerId, seeder.MyPeerId)
	}

	rand.Read(stringLength)

	protocolString = make([]byte, stringLength[0])

	rand.Read(protocolString)
	rand.Read(peerId)
	rand.Read(extensions)

	if _, err := exteriorConn.Write(stringLength); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(protocolString); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(extensions); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(infoHash); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(peerId); err != nil {
		panic(err)
	}

	wait.Wait()

	if bytes.Compare(seeder.PeerId, peerId) != 0 {
		t.Errorf("Peer id is wrong: %v != %v", peerId, seeder.PeerId)
	}

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
		if err != nil {
			t.Error(err)
		}
	}()

	stringLength := make([]byte, 1)
	peerId := make([]byte, 20)
	extensions := make([]byte, 8)

	rand.Read(stringLength)

	protocolString := make([]byte, stringLength[0])

	rand.Read(protocolString)
	rand.Read(peerId)
	rand.Read(extensions)

	if _, err := exteriorConn.Write(stringLength); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(protocolString); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(extensions); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(infoHash); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(peerId); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Read(stringLength); err != nil {
		panic(err)
	}

	if stringLength[0] != 19 {
		t.Errorf("String length is wrong: %d", stringLength[0])
	}

	protocolString = make([]byte, 19)

	if _, err := exteriorConn.Read(protocolString); err != nil {
		panic(err)
	}

	if bytes.Compare(protocolString, []byte("BitTorrent protocol")) != 0 {
		t.Errorf("Protocol tring is wrong: %s", string(protocolString))
	}

	if _, err := exteriorConn.Read(extensions); err != nil {
		panic(err)
	}

	if bytes.Compare(extensions, []byte{0, 0, 0, 0, 0, 0, 0, 0}) != 0 {
		t.Errorf("Extension bits is not null: %v", extensions)
	}

	if _, err := exteriorConn.Read(infoHash); err != nil {
		panic(err)
	}

	if bytes.Compare(infoHash, seeder.InfoHash) != 0 {
		t.Errorf("Info hash is wrong: %v != %v", infoHash, seeder.InfoHash)
	}

	if _, err := exteriorConn.Read(myPeerId); err != nil {
		panic(err)
	}

	if bytes.Compare(myPeerId, seeder.MyPeerId) != 0 {
		t.Errorf("My peer id is wrong: %v != %v", myPeerId, seeder.MyPeerId)
	}

	wait.Wait()

	if bytes.Compare(seeder.PeerId, peerId) != 0 {
		t.Errorf("Peer id is wrong: %v != %v", peerId, seeder.PeerId)
	}

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
		if err == nil {
			t.Error(err)
		}
	}()

	timer := time.AfterFunc(time.Second*1, func() {
		t.Errorf("IO blocked, error wasn't appear.")
		_ = exteriorConn.Close()
		_ = interiorConn.Close()
	})

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

	if _, err := exteriorConn.Write(stringLength); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(protocolString); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(extensions); err != nil {
		panic(err)
	}

	if _, err := exteriorConn.Write(wrongInfoHash); err != nil {
		panic(err)
	}

	wait.Wait()

	timer.Stop()
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
		if err != nil {
			t.Error(err)
		}
	}()

	go func() {
		defer wait.Done()
		err := secondSeeder.Accept(secondConn)
		if err != nil {
			t.Error(err)
		}
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

	if message.Id != KeepAlive || message.Payload != nil {
		t.Errorf("Message is corrupt: id=%d, payload=%v", message.Id, message.Payload)
	}

	firstSeeder.outcoming <- Message{Interested, nil, nil}
	message = <-secondSeeder.incoming

	if message.Id != Interested || message.Payload != nil {
		t.Errorf("Message is corrupt: id=%d, payload=%v", message.Id, message.Payload)
	}

	payload := make([]byte, 215)
	rand.Read(payload)

	firstSeeder.outcoming <- Message{Bitfield, payload, nil}
	message = <-secondSeeder.incoming

	if message.Id != Bitfield || bytes.Compare(message.Payload, payload) != 0 {
		t.Errorf("Message is corrupt: id=%d, payload=%v", message.Id, message.Payload)
	}

	firstSeeder.Close()
	secondSeeder.Close()

	wait.Wait()

}

func TestSeeder_MakePayload(t *testing.T) {

	pieceIndex := rand.Uint32()
	offset := rand.Uint32()
	length := rand.Uint32()

	payload := MakeHavePayload(pieceIndex)
	parsedPieceIndex := binary.BigEndian.Uint32(payload)
	if parsedPieceIndex != pieceIndex {
		t.Errorf("Have payload is wrong: %d != %d",
			pieceIndex, parsedPieceIndex)
	}

	payload = MakeRequestPayload(pieceIndex, offset, length)
	parsedPieceIndex = binary.BigEndian.Uint32(payload[0:4])
	parsedOffset := binary.BigEndian.Uint32(payload[4:8])
	parsedLength := binary.BigEndian.Uint32(payload[8:12])
	if parsedPieceIndex != pieceIndex || parsedOffset != offset || parsedLength != length {
		t.Errorf("Request payload is wrong: (%d, %d, %d) != (%d, %d, %d)",
			pieceIndex, offset, length, parsedPieceIndex, parsedOffset, parsedLength)
	}

	payload = MakeCancelPayload(pieceIndex, offset, length)
	parsedPieceIndex = binary.BigEndian.Uint32(payload[0:4])
	parsedOffset = binary.BigEndian.Uint32(payload[4:8])
	parsedLength = binary.BigEndian.Uint32(payload[8:12])
	if parsedPieceIndex != pieceIndex || parsedOffset != offset || parsedLength != length {
		t.Errorf("Cancel payload is wrong: (%d, %d, %d) != (%d, %d, %d)",
			pieceIndex, offset, length, parsedPieceIndex, parsedOffset, parsedLength)
	}

	payload = MakeCancelPayload(pieceIndex, offset, length)
	parsedPieceIndex = binary.BigEndian.Uint32(payload[0:4])
	parsedOffset = binary.BigEndian.Uint32(payload[4:8])
	parsedLength = binary.BigEndian.Uint32(payload[8:12])
	if parsedPieceIndex != pieceIndex || parsedOffset != offset || parsedLength != length {
		t.Errorf("Cancel payload is wrong: (%d, %d, %d) != (%d, %d, %d)",
			pieceIndex, offset, length, parsedPieceIndex, parsedOffset, parsedLength)
	}

	block := make([]byte, 256)
	rand.Read(block)

	payload = MakePiecePayload(pieceIndex, offset, block)
	parsedPieceIndex = binary.BigEndian.Uint32(payload[0:4])
	parsedOffset = binary.BigEndian.Uint32(payload[4:8])
	parsedBlock := payload[8:]
	if parsedPieceIndex != pieceIndex || parsedOffset != offset || bytes.Compare(parsedBlock, block) != 0 {
		t.Errorf("Piece payload is wrong: (%d, %d, %v) != (%d, %d, %v)",
			pieceIndex, offset, block, parsedPieceIndex, parsedOffset, parsedBlock)
	}
}

func TestSeeder_MakeAndParsePayload(t *testing.T) {

	pieceIndex := rand.Uint32()
	offset := rand.Uint32()
	length := rand.Uint32()

	payload := MakeHavePayload(pieceIndex)
	parsedPieceIndex, err := ParseHavePayload(payload)

	if err != nil {
		t.Error("Can not parse have payload:", err)
	}

	if parsedPieceIndex != pieceIndex {
		t.Errorf("Make and parse for have doesnt correspond: %d != %d",
			pieceIndex, parsedPieceIndex)
	}

	_, err = ParseHavePayload(payload[1:])
	if err == nil {
		t.Errorf("Parse have payload with wrong length %d", len(payload[1:]))
	}

	payload = MakeRequestPayload(pieceIndex, offset, length)
	parsedPieceIndex, parsedOffset, parsedLength, err := ParseRequestPayload(payload)

	if err != nil {
		t.Error("Can not parse request payload:", err)
	}

	if parsedPieceIndex != pieceIndex || parsedOffset != offset || parsedLength != length {
		t.Errorf("Make and parse for request doesnt correspond: (%d, %d, %d) != (%d, %d, %d)",
			pieceIndex, offset, length, parsedPieceIndex, parsedOffset, parsedLength)
	}

	_, _, _, err = ParseRequestPayload(payload[1:])
	if err == nil {
		t.Errorf("Parse request payload with wrong length %d", len(payload[1:]))
	}

	payload = MakeCancelPayload(pieceIndex, offset, length)
	parsedPieceIndex, parsedOffset, parsedLength, err = ParseCancelPayload(payload)

	if err != nil {
		t.Error("Can not parse cancel payload:", err)
	}

	if parsedPieceIndex != pieceIndex || parsedOffset != offset || parsedLength != length {
		t.Errorf("Make and parse for cancel doesnt correspond: (%d, %d, %d) != (%d, %d, %d)",
			pieceIndex, offset, length, parsedPieceIndex, parsedOffset, parsedLength)
	}

	_, _, _, err = ParseCancelPayload(payload[1:])
	if err == nil {
		t.Errorf("Parse cancel payload with wrong length %d", len(payload[1:]))
	}

	block := make([]byte, 256)
	rand.Read(block)

	payload = MakePiecePayload(pieceIndex, offset, block)
	parsedPieceIndex, parsedOffset, parsedBlock, err := ParsePiecePayload(payload)

	if err != nil {
		t.Error("Can not parse piece payload:", err)
	}

	if parsedPieceIndex != pieceIndex || parsedOffset != offset || bytes.Compare(parsedBlock, block) != 0 {
		t.Errorf("Make and parse for piece doesnt correspond: (%d, %d, %v) != (%d, %d, %v)",
			pieceIndex, offset, block, parsedPieceIndex, parsedOffset, parsedBlock)
	}

	_, _, _, err = ParsePiecePayload(payload[:7])
	if err == nil {
		t.Errorf("Parse cancel payload with invalid length %d", len(payload[:7]))
	}

}
