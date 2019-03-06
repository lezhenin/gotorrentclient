package tracker

import (
	"encoding/binary"
	"fmt"
	"github.com/lezhenin/gotorrentclient/bitfield"
	"io"
	"net"
)

const blockSize int = 16 * 1024
const bufferSize = blockSize + 16

type MessageId uint8

const (
	KeepAlive     MessageId = 255 // has no id
	Choke         MessageId = 0
	Unchoke       MessageId = 1
	Interested    MessageId = 2
	NotInterested MessageId = 3
	Have          MessageId = 4
	Bitfield      MessageId = 5
	Request       MessageId = 6
	Piece         MessageId = 7
	Cancel        MessageId = 8
)

type Seeder struct {
	NetAddress net.Addr

	Complete bool

	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool

	PeerBitfield bitfield.Bitfield
	PeerId       []byte

	MyPeerId []byte
	InfoHash []byte

	connectionTCP io.ReadWriter
	buffer        []byte
}

func NewSeeder(addr string, infoHash []byte, peerId []byte) (seeder *Seeder, err error) {

	seeder = new(Seeder)
	seeder.NetAddress, err = net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	seeder.AmChoking = true
	seeder.PeerChoking = true

	seeder.AmInterested = false
	seeder.PeerInterested = false

	seeder.MyPeerId = peerId
	seeder.InfoHash = infoHash

	seeder.buffer = make([]byte, bufferSize)

	return seeder, nil

}

func (s *Seeder) SendHandshakeMessage() (err error) {

	fmt.Printf("connect to %s\n", s.NetAddress.String())

	s.connectionTCP, err = net.Dial(s.NetAddress.Network(), s.NetAddress.String())
	if err != nil {
		return err
	}

	fmt.Printf("send to %s\n", s.NetAddress.String())

	message := makeHandshakeMessage(s.InfoHash, s.MyPeerId)
	_, err = s.connectionTCP.Write(message)
	if err != nil {
		return err
	}

	return nil

}

func makeHandshakeMessage(infoHash []byte, peerId []byte) (message []byte) {

	protocolString := "BitTorrent protocol"

	message = make([]byte, 49+19)

	message[0] = 19
	copy(message[1:20], []byte(protocolString))
	copy(message[28:48], infoHash)
	copy(message[48:68], peerId)

	binary.BigEndian.PutUint64(message[20:28], 0)

	return message
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

	copy(block, payload[8:])

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

func makeMessage(id MessageId, payload []byte) (message []byte) {

	if id == KeepAlive {
		message = make([]byte, 1)
		message[0] = 0
		return message
	}

	length := 1 + len(payload)
	message = make([]byte, length+1)
	message[0] = byte(length)
	message[1] = byte(id)
	copy(message[2:length], payload)

	return message
}

func parseMessage(message []byte) (id MessageId, payload []byte, err error) {

	if len(message) == 0 {
		return 0, nil,
			fmt.Errorf("parse message: byte slice is empty")
	}

	length := int(message[0])

	if length+1 != len(message) {
		return 0, nil,
			fmt.Errorf("parse message: byte slice length and message length are not equal")
	}

	if length == 0 {
		return KeepAlive, nil, nil
	}

	id = MessageId(message[1])

	if id > Cancel && id != KeepAlive {
		return 0, nil,
			fmt.Errorf("parse message: message has unsupported id %d", id)
	}

	payload = make([]byte, length-1)
	copy(payload, message[2:])

	return id, payload, nil
}
