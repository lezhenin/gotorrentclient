package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/lezhenin/gotorrentclient/bitfield"
	"io"
	"log"
	"net"
	"time"
)

const protocolId string = "BitTorrent protocol"

const handshakeTimeout = 15
const requestTimeout = 15

const keepAliveTimeout = 120

const bufferSize = blockLength + 512
const messageBufferLength = 16

type MessageId uint8

const (
	Choke         MessageId = 0
	Unchoke       MessageId = 1
	Interested    MessageId = 2
	NotInterested MessageId = 3
	Have          MessageId = 4
	Bitfield      MessageId = 5
	Request       MessageId = 6
	Piece         MessageId = 7
	Cancel        MessageId = 8
	KeepAlive     MessageId = 255 // has no id
	Error         MessageId = 254 // special code
)

type Message struct {
	Id      MessageId
	Payload []byte
	PeerId  []byte
}

type Seeder struct {
	NetAddress net.Addr

	Complete bool

	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool

	PeerBitfield *bitfield.Bitfield
	MyBitfield   *bitfield.Bitfield

	PeerId []byte

	MyPeerId []byte
	InfoHash []byte

	connection net.Conn
	buffer     []byte

	incoming  chan Message
	outcoming chan Message

	keepAlive      bool
	keepAliveTimer *time.Timer
}

func (s *Seeder) Accept(connection net.Conn) (err error) {

	s.connection = connection

	// set deadline 15 second for handshake
	err = s.connection.SetDeadline(time.Now().Add(handshakeTimeout * time.Second))
	if err != nil {
		return err
	}

	s.PeerId, err = readHandshakeMessage(s.connection, s.InfoHash)
	if err != nil {
		return err
	}

	err = writeHandshakeMessage(s.connection, s.InfoHash, s.MyPeerId)
	if err != nil {
		return err
	}

	// if handshake done clear deadline
	err = s.connection.SetDeadline(time.Time{})
	if err != nil {
		return err
	}

	return nil

}

func (s *Seeder) Init(connection net.Conn) (err error) {

	s.connection = connection

	// set deadline 15 second for handshake
	err = s.connection.SetDeadline(time.Now().Add(handshakeTimeout * time.Second))
	if err != nil {
		return err
	}

	err = writeHandshakeMessage(s.connection, s.InfoHash, s.MyPeerId)
	if err != nil {
		return err
	}

	s.PeerId, err = readHandshakeMessage(s.connection, s.InfoHash)
	if err != nil {
		return err
	}

	// if handshake done clear deadline
	err = s.connection.SetDeadline(time.Time{})
	if err != nil {
		return err
	}

	return nil

}

func (s *Seeder) Run() {

	s.keepAliveTimer = time.NewTimer(time.Second * keepAliveTimeout)

	go s.keepAliveRoutine()
	go s.readRoutine()
	go s.writeRoutine()

}

func (s *Seeder) keepAliveRoutine() {

	for {

		<-s.keepAliveTimer.C
		s.outcoming <- Message{KeepAlive, nil, s.MyPeerId}
		s.keepAliveTimer.Reset(time.Second * keepAliveTimeout)

	}

}

func (s *Seeder) readRoutine() {

	for {

		id, payload, err := readMessage(s.connection)

		if err != nil {
			log.Println(err)
			s.incoming <- Message{Error, []byte(err.Error()), s.PeerId}
			return
		}

		if id == Piece {
			err = s.connection.SetReadDeadline(time.Time{})
			if err != nil {
				log.Println(err)
				s.incoming <- Message{Error, []byte(err.Error()), s.PeerId}
				return
			}
		}

		s.incoming <- Message{id, payload, s.PeerId}

	}
}

func (s *Seeder) writeRoutine() {

	for {

		message := <-s.outcoming

		if message.Id == Request {
			err := s.connection.SetReadDeadline(time.Now().Add(requestTimeout * time.Second))
			if err != nil {
				log.Println(err)
				s.incoming <- Message{Error, []byte(err.Error()), s.PeerId}
				return
			}
		}

		err := writeMessage(s.connection, message.Id, message.Payload)
		if err != nil {
			log.Println(err)
			s.incoming <- Message{Error, []byte(err.Error()), s.PeerId}
			return
		}

		s.keepAliveTimer.Reset(time.Second * keepAliveTimeout)

	}
}

func (s *Seeder) Close() {

	if s.keepAliveTimer != nil {
		s.keepAliveTimer.Stop()
	}
	close(s.outcoming)
	_ = s.connection.Close()

}

func NewSeeder(infoHash []byte, peerId []byte, incoming chan Message) (seeder *Seeder, err error) {

	seeder = new(Seeder)

	seeder.AmChoking = true
	seeder.PeerChoking = true

	seeder.AmInterested = false
	seeder.PeerInterested = false

	seeder.MyPeerId = peerId
	seeder.InfoHash = infoHash

	seeder.buffer = make([]byte, bufferSize)

	seeder.incoming = incoming
	seeder.outcoming = make(chan Message, messageBufferLength)

	return seeder, nil

}

func writeHandshakeMessage(w io.Writer, infoHash []byte, peerId []byte) (err error) {

	protocolString := protocolId

	message := make([]byte, 49+19)

	message[0] = 19
	copy(message[1:20], []byte(protocolString))
	copy(message[28:48], infoHash)
	copy(message[48:68], peerId)

	binary.BigEndian.PutUint64(message[20:28], 0)

	_, err = w.Write(message)
	if err != nil {
		return err
	}

	return nil
}

func readHandshakeMessage(r io.Reader, expectedInfoHash []byte) (peerId []byte, err error) {

	stringLengthBuffer := make([]byte, 1)

	_, err = io.ReadAtLeast(r, stringLengthBuffer, 1)
	if err != nil {
		return nil, err
	}

	stringLength := uint8(stringLengthBuffer[0])

	protocolStringBuffer := make([]byte, stringLength)
	_, err = io.ReadFull(r, protocolStringBuffer)
	if err != nil {
		return nil, err
	}
	protocolString := string(protocolStringBuffer)

	extensionBuffer := make([]byte, 8)
	_, err = io.ReadFull(r, extensionBuffer)
	if err != nil {
		return nil, err
	}

	infoHash := make([]byte, 20)
	_, err = io.ReadFull(r, infoHash)
	if err != nil {
		return nil, err
	}

	if bytes.Compare(expectedInfoHash, infoHash) != 0 {
		return nil, fmt.Errorf("parse handshake message:" +
			" info hash doesn't match")
	}

	peerId = make([]byte, 20)
	_, err = io.ReadFull(r, peerId)
	if err != nil {
		return nil, err
	}

	log.Printf("Handshake received: protocol %s, peer id = %v, info hash = %v",
		protocolString, peerId, infoHash)

	return peerId, nil

}

func makeHavePayload(pieceIndex uint32) (payload []byte) {

	payload = make([]byte, 4)
	binary.BigEndian.PutUint32(payload, pieceIndex)
	return payload
}

func parseHavePayload(payload []byte) (pieceIndex uint32, err error) {

	if len(payload) != 4 {
		return 0,
			fmt.Errorf("parse have payload: byte slice has wrong length %d", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload)
	return pieceIndex, nil
}

func makeRequestPayload(pieceIndex uint32, begin uint32, length uint32) (payload []byte) {

	payload = make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return payload
}

func parseRequestPayload(payload []byte) (pieceIndex uint32, begin uint32, length uint32, err error) {

	if len(payload) != 12 {
		return 0, 0, 0,
			fmt.Errorf("parse request payload: byte slice has wrong length %d", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload[0:4])
	begin = binary.BigEndian.Uint32(payload[4:8])
	length = binary.BigEndian.Uint32(payload[8:12])

	return pieceIndex, begin, length, nil
}

func makePiecePayload(pieceIndex uint32, begin uint32, block []byte) (payload []byte) {

	payload = make([]byte, 8+len(block))
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	copy(payload[8:(8+len(block))], block)

	return payload
}

func parsePiecePayload(payload []byte) (pieceIndex uint32, begin uint32, block []byte, err error) {

	if len(payload) < 8 {
		return 0, 0, nil,
			fmt.Errorf("parse piece payload: byte slice length %d less than 9", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload[0:4])
	begin = binary.BigEndian.Uint32(payload[4:8])

	block = payload[8:]

	return pieceIndex, begin, block, nil
}

func makeCancelPayload(pieceIndex uint32, begin uint32, length uint32) (payload []byte) {

	payload = make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return payload
}

func parseCancelPayload(payload []byte) (pieceIndex uint32, begin uint32, length uint32, err error) {

	if len(payload) != 12 {
		return 0, 0, 0,
			fmt.Errorf("parse cancel payload: byte slice has wrong length %d", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload[0:4])
	begin = binary.BigEndian.Uint32(payload[4:8])
	length = binary.BigEndian.Uint32(payload[8:12])

	return pieceIndex, begin, length, nil
}

func writeMessage(w io.Writer, id MessageId, payload []byte) (err error) {

	var message []byte

	if id == KeepAlive {
		message = make([]byte, 4)
		binary.BigEndian.PutUint32(message[0:4], 0)
		return nil
	}

	length := 1 + len(payload)
	message = make([]byte, length+4)
	binary.BigEndian.PutUint32(message[0:4], uint32(length))
	message[4] = byte(id)
	copy(message[5:(5+length-1)], payload)

	_, err = w.Write(message)
	if err != nil {
		return err
	}

	//log.Printf("Message sent: id = %d, len = %d", id, length)

	return nil
}

func readMessage(r io.Reader) (id MessageId, payload []byte, err error) {

	lengthBuffer := make([]byte, 4)

	_, err = io.ReadAtLeast(r, lengthBuffer, 4)
	if err != nil {
		return Error, nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuffer)

	if length == 0 {
		return KeepAlive, nil, nil
	}

	idBuffer := make([]byte, 1)
	_, err = io.ReadAtLeast(r, idBuffer, 1)
	if err != nil {
		panic(err)
	}

	id = MessageId(idBuffer[0])

	if id > Cancel && id != KeepAlive {
		return Error, nil,
			fmt.Errorf("read message: message has unsupported id %d", id)
	}

	if length == 1 {
		return id, nil, nil
	}

	payload = make([]byte, length-1)
	_, err = io.ReadFull(r, payload)

	if err != nil {
		return Error, nil, err
	}

	//log.Printf("Message received: id = %d, len = %d", id, length)

	return id, payload, nil

}
