package torrent

import (
	"bytes"
	"encoding/binary"
	"github.com/juju/errors"
	"github.com/lezhenin/gotorrentclient/pkg/bitfield"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
	"time"
)

const protocolId string = "BitTorrent protocol"

const handshakeTimeout = 15
const requestTimeout = 15

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
	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool

	PeerBitfield *bitfield.Bitfield

	PeerId   []byte
	MyPeerId []byte

	InfoHash []byte

	connection net.Conn
	buffer     []byte

	incoming  chan Message
	outcoming chan Message

	closeGroup          sync.WaitGroup
	closeConnectionOnce sync.Once
	closeChannelsOnce   sync.Once
	closeChan           chan struct{}
}

func (s *Seeder) Accept(connection net.Conn) (err error) {

	s.connection = connection

	// set deadline 15 second for handshake
	err = s.connection.SetDeadline(time.Now().Add(handshakeTimeout * time.Second))
	if err != nil {
		return errors.Annotate(err, "accept new seeder connection")
	}

	s.PeerId, err = s.readHandshakeMessage()
	if err != nil {
		return errors.Annotate(err, "accept new seeder connection")
	}

	seederLogger.WithFields(logrus.Fields{
		"peer":     s.PeerId,
		"infoHash": s.InfoHash,
	}).Trace("Handshake received")

	err = s.writeHandshakeMessage()
	if err != nil {
		return errors.Annotate(err, "accept new seeder connection")
	}

	seederLogger.WithFields(logrus.Fields{
		"peer":     s.PeerId,
		"infoHash": s.InfoHash,
	}).Trace("Handshake sent")

	// if handshake done clear deadline
	err = s.connection.SetDeadline(time.Time{})
	if err != nil {
		return errors.Annotate(err, "accept new seeder connection")
	}

	return nil

}

func (s *Seeder) Dial(connection net.Conn) (err error) {

	s.connection = connection

	// set deadline 15 second for handshake
	err = s.connection.SetDeadline(time.Now().Add(handshakeTimeout * time.Second))
	if err != nil {
		return errors.Annotate(err, "init new seeder connection")
	}

	err = s.writeHandshakeMessage()
	if err != nil {
		return errors.Annotate(err, "init new seeder connection")
	}

	seederLogger.WithFields(logrus.Fields{
		"peer":     s.PeerId,
		"infoHash": s.InfoHash,
	}).Trace("Handshake sent")

	s.PeerId, err = s.readHandshakeMessage()
	if err != nil {
		return errors.Annotate(err, "init new seeder connection")
	}

	seederLogger.WithFields(logrus.Fields{
		"peer":     s.PeerId,
		"infoHash": s.InfoHash,
	}).Trace("Handshake received")

	// if handshake done clear deadline
	err = s.connection.SetDeadline(time.Time{})
	if err != nil {
		return errors.Annotate(err, "init new seeder connection")
	}

	return nil

}

func (s *Seeder) Start() {

	s.closeGroup.Add(2)

	go func() {
		s.read()
		s.closeConnectionOnce.Do(s.closeConnection)
		s.closeGroup.Done()
	}()

	go func() {
		s.write()
		s.closeConnectionOnce.Do(s.closeConnection)
		s.closeGroup.Done()
	}()

	seederLogger.WithFields(logrus.Fields{
		"peerId":   s.PeerId,
		"infoHash": s.InfoHash,
	}).Info("seeder run")

	s.closeGroup.Wait()
}

func (s *Seeder) Close() {

	seederLogger.WithFields(logrus.Fields{
		"peerId":   s.PeerId,
		"infoHash": s.InfoHash,
	}).Info("seeder closed")

	s.closeConnectionOnce.Do(s.closeConnection)
	s.closeGroup.Wait()
	s.closeChannelsOnce.Do(s.closeChannels)
}

func (s *Seeder) closeChannels() {

	close(s.closeChan)
	close(s.outcoming)
}

func (s *Seeder) closeConnection() {

	seederLogger.WithFields(logrus.Fields{
		"peerId":   s.PeerId,
		"infoHash": s.InfoHash,
	}).Debug("seeder connection closed")

	s.closeChan <- struct{}{}
	s.closeChan <- struct{}{}

	_ = s.connection.Close()
}

func (s *Seeder) read() {

	for {

		id, payload, err := s.readMessage()
		if err != nil {
			return
		}

		if id == Piece {
			err = s.connection.SetReadDeadline(time.Time{})
			if err != nil {
				return
			}
		}

		seederLogger.WithFields(logrus.Fields{
			"id":       id,
			"len":      len(payload),
			"peerId":   s.PeerId,
			"infoHash": s.InfoHash,
		}).Trace("Message received")

		select {
		case s.incoming <- Message{id, payload, s.PeerId}:
			continue
		case <-s.closeChan:
			return
		}
	}
}

func (s *Seeder) write() {

	for {
		select {
		case message := <-s.outcoming:

			if message.Id == Request {
				err := s.connection.SetReadDeadline(time.Now().Add(requestTimeout * time.Second))
				if err != nil {
					seederLogger.WithFields(logrus.Fields{
						"infoHash": s.InfoHash,
						"peerId":   s.PeerId,
					}).Error(errors.Annotate(err, "seeder write"))
					return
				}
			}

			err := s.writeMessage(message.Id, message.Payload)
			if err != nil {
				seederLogger.WithFields(logrus.Fields{
					"infoHash": s.InfoHash,
					"peerId":   s.PeerId,
				}).Error(errors.Annotate(err, "seeder write"))
				return
			}

			seederLogger.WithFields(logrus.Fields{
				"id":       message.Id,
				"len":      len(message.Payload),
				"peerId":   s.PeerId,
				"infoHash": s.InfoHash,
			}).Trace("Message sent")

		case <-s.closeChan:
			return
		}
	}
}

func (s *Seeder) writeMessage(id MessageId, payload []byte) (err error) {

	var message []byte

	if id == KeepAlive {
		message = make([]byte, 4)
		binary.BigEndian.PutUint32(message[0:4], 0)
		_, err = s.connection.Write(message)
		if err != nil {
			return errors.Annotate(err, "write message")
		}
		return nil
	}

	length := 1 + len(payload)
	message = make([]byte, length+4)
	binary.BigEndian.PutUint32(message[0:4], uint32(length))
	message[4] = byte(id)
	copy(message[5:(5+length-1)], payload)

	_, err = s.connection.Write(message)
	if err != nil {
		return errors.Annotate(err, "write message")
	}

	return nil
}

func (s *Seeder) readMessage() (id MessageId, payload []byte, err error) {

	lengthBuffer := make([]byte, 4)

	_, err = io.ReadAtLeast(s.connection, lengthBuffer, 4)
	if err != nil {
		return Error, nil, errors.Annotate(err, "read message")
	}

	length := binary.BigEndian.Uint32(lengthBuffer)

	if length == 0 {
		return KeepAlive, nil, nil
	}

	idBuffer := make([]byte, 1)
	_, err = io.ReadAtLeast(s.connection, idBuffer, 1)
	if err != nil {
		return Error, nil, errors.Annotate(err, "read message")
	}

	id = MessageId(idBuffer[0])

	if id > Cancel && id != KeepAlive {
		return Error, nil,
			errors.Errorf("read message: message has unknown id %d", id)
	}

	if length == 1 {
		return id, nil, nil
	}

	payload = make([]byte, length-1)
	_, err = io.ReadFull(s.connection, payload)
	if err != nil {
		return Error, nil, errors.Annotate(err, "read message")
	}

	return id, payload, nil

}

func (s *Seeder) writeHandshakeMessage() (err error) {

	protocolString := protocolId

	message := make([]byte, 49+19)

	message[0] = 19
	copy(message[1:20], []byte(protocolString))
	copy(message[28:48], s.InfoHash)
	copy(message[48:68], s.MyPeerId)

	binary.BigEndian.PutUint64(message[20:28], 0)

	_, err = s.connection.Write(message)
	if err != nil {
		return errors.Annotate(err, "write handshake message")
	}

	return nil
}

func (s *Seeder) readHandshakeMessage() (peerId []byte, err error) {

	stringLengthBuffer := make([]byte, 1)

	_, err = io.ReadAtLeast(s.connection, stringLengthBuffer, 1)
	if err != nil {
		return nil, errors.Annotate(err, "read handshake message")
	}

	stringLength := uint8(stringLengthBuffer[0])

	protocolStringBuffer := make([]byte, stringLength)
	_, err = io.ReadFull(s.connection, protocolStringBuffer)
	if err != nil {
		return nil, errors.Annotate(err, "read handshake message")
	}
	//protocolString := string(protocolStringBuffer)

	extensionBuffer := make([]byte, 8)
	_, err = io.ReadFull(s.connection, extensionBuffer)
	if err != nil {
		return nil, errors.Annotate(err, "read handshake message")
	}

	infoHash := make([]byte, 20)
	_, err = io.ReadFull(s.connection, infoHash)
	if err != nil {
		return nil, errors.Annotate(err, "read handshake message")
	}

	if bytes.Compare(s.InfoHash, infoHash) != 0 {
		return nil,
			errors.Errorf("read handshake message: info hash doesn't match")
	}

	peerId = make([]byte, 20)
	_, err = io.ReadFull(s.connection, peerId)
	if err != nil {
		return nil, errors.Annotate(err, "read handshake message")
	}

	return peerId, nil

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

	seeder.closeChan = make(chan struct{}, 2)

	return seeder, nil

}

func MakeHavePayload(pieceIndex uint32) (payload []byte) {

	payload = make([]byte, 4)
	binary.BigEndian.PutUint32(payload, pieceIndex)
	return payload
}

func ParseHavePayload(payload []byte) (pieceIndex uint32, err error) {

	if len(payload) != 4 {
		return 0,
			errors.Errorf("parse have payload: byte slice has wrong length %d", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload)
	return pieceIndex, nil
}

func MakeRequestPayload(pieceIndex uint32, begin uint32, length uint32) (payload []byte) {

	payload = make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return payload
}

func ParseRequestPayload(payload []byte) (pieceIndex uint32, begin uint32, length uint32, err error) {

	if len(payload) != 12 {
		return 0, 0, 0,
			errors.Errorf("parse request payload: message has unexpected length %d", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload[0:4])
	begin = binary.BigEndian.Uint32(payload[4:8])
	length = binary.BigEndian.Uint32(payload[8:12])

	return pieceIndex, begin, length, nil
}

func MakePiecePayload(pieceIndex uint32, begin uint32, block []byte) (payload []byte) {

	payload = make([]byte, 8+len(block))
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	copy(payload[8:(8+len(block))], block)

	return payload
}

func ParsePiecePayload(payload []byte) (pieceIndex uint32, begin uint32, block []byte, err error) {

	if len(payload) < 8 {
		return 0, 0, nil,
			errors.Errorf("parse piece payload: message has length %d less than 9", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload[0:4])
	begin = binary.BigEndian.Uint32(payload[4:8])

	block = payload[8:]

	return pieceIndex, begin, block, nil
}

func MakeCancelPayload(pieceIndex uint32, begin uint32, length uint32) (payload []byte) {

	payload = make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return payload
}

func ParseCancelPayload(payload []byte) (pieceIndex uint32, begin uint32, length uint32, err error) {

	if len(payload) != 12 {
		return 0, 0, 0,
			errors.Errorf("parse cancel payload: message has unexpected length %d", len(payload))
	}

	pieceIndex = binary.BigEndian.Uint32(payload[0:4])
	begin = binary.BigEndian.Uint32(payload[4:8])
	length = binary.BigEndian.Uint32(payload[8:12])

	return pieceIndex, begin, length, nil
}
