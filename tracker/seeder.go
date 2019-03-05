package tracker

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Seeder struct {
	NetAddress net.Addr
	Complete   bool

	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool

	PeerId   []byte
	InfoHash []byte

	connectionTCP io.ReadWriter
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

	seeder.PeerId = peerId
	seeder.InfoHash = infoHash

	return seeder, nil

}

func (s *Seeder) SendHandshakeMessage() (err error) {

	fmt.Printf("connect to %s\n", s.NetAddress.String())

	s.connectionTCP, err = net.Dial(s.NetAddress.Network(), s.NetAddress.String())
	if err != nil {
		return err
	}

	fmt.Printf("send to %s\n", s.NetAddress.String())

	message := makeHandshakeMessage(s.InfoHash, s.PeerId)
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
