package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
)

type Connection struct {
	Tracker       Tracker
	Download      Download
	ConnectionUDP io.ReadWriter
	ConnectionId  uint64
	Established   bool
}

func (connection *Connection) EstablishConnection() (err error) {

	transactionId := rand.Uint32()

	fmt.Println(transactionId)

	connection.ConnectionUDP, err = net.Dial("udp", "opentor.org:2710")

	println(connection.ConnectionUDP)

	if err != nil {
		panic(err)
	}

	_, err = connection.writeConnectionRequest(transactionId)

	if err != nil {
		panic(err)
	}

	err = connection.readConnectionResponse(transactionId)

	if err != nil {
		panic(err)
	}

	connection.Established = true

	return nil

}

func (connection *Connection) writeConnectionRequest(transactionId uint32) (int, error) {

	data := make([]byte, 16)

	protocolIdBytes := data[0:8]
	actionBytes := data[8:12]
	transactionIdBytes := data[12:16]

	binary.BigEndian.PutUint64(protocolIdBytes, 0x41727101980)
	binary.BigEndian.PutUint32(actionBytes, 0)
	binary.BigEndian.PutUint32(transactionIdBytes, transactionId)

	return connection.ConnectionUDP.Write(data)

}

func (connection *Connection) readConnectionResponse(transactionId uint32) (err error) {

	data := make([]byte, 16)

	n, err := connection.ConnectionUDP.Read(data)

	println(n)

	if err != nil {
		panic(err)
	}

	actionBytes := data[0:4]
	transactionIdBytes := data[4:8]
	connectionIdBytes := data[8:16]

	if binary.BigEndian.Uint32(transactionIdBytes) != transactionId {
		panic("Transaction id is wrong")
	}

	if binary.BigEndian.Uint32(actionBytes) != 0 {
		panic("Action is not null")
	}

	connection.ConnectionId = binary.BigEndian.Uint64(connectionIdBytes)

	return nil

}

type Tracker struct {
	Announce string
}

//func (tracker *Tracker) GetSeeders (n int32) {
//
//
//
//}

//func MakeUDPAnounceRequest(download Download) []byte {
//
//
//
//}
