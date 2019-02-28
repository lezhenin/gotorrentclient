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

//Offset  Size    Name    Value
//0       64-bit integer  connection_id
//8       32-bit integer  action          1 // announce
//12      32-bit integer  transaction_id
//16      20-byte string  info_hash
//36      20-byte string  peer_id
//56      64-bit integer  downloaded
//64      64-bit integer  left
//72      64-bit integer  uploaded
//80      32-bit integer  event           0 // 0: none; 1: completed; 2: started; 3: stopped
//84      32-bit integer  IP address      0 // default
//88      32-bit integer  key
//92      32-bit integer  num_want        -1 // default
//96      16-bit integer  port
//98

func (connection *Connection) writeAnnounceRequest(transactionId uint32) (int, error) {

	data := make([]byte, 96)

	connectionIdBytes := data[0:8]
	actionBytes := data[8:12]
	transactionIdBytes := data[12:16]
	infoHashBytes := data[16:36]
	peerIdBytes := data[36:56]
	downloadedBytes := data[56:64]
	leftBytes := data[64:72]
	uploadedBytes := data[72:80]
	eventBytes := data[80:84]
	addressIpBytes := data[84:88]
	keyBytes := data[88:92]
	numWantBytes := data[92:96]
	portBytes := data[96:98]

	binary.BigEndian.PutUint64(connectionIdBytes, connection.ConnectionId)
	binary.BigEndian.PutUint32(actionBytes, 1)
	binary.BigEndian.PutUint32(transactionIdBytes, transactionId)
	binary.BigEndian.PutUint64(downloadedBytes, connection.Download.Downloaded)
	binary.BigEndian.PutUint64(leftBytes, connection.Download.Left)
	binary.BigEndian.PutUint64(uploadedBytes, connection.Download.Uploaded)
	binary.BigEndian.PutUint32(eventBytes, 0)     // todo
	binary.BigEndian.PutUint32(addressIpBytes, 0) // default
	binary.BigEndian.PutUint32(keyBytes, 0)
	binary.BigEndian.PutUint32(numWantBytes, -1) // default
	binary.BigEndian.PutUint16(portBytes, connection.Download.Port)

	infoHashBytes[:] = connection.Download.Metadata.Info.HashSHA1
	peerIdBytes[:] = connection.Download.PeerId

	return connection.ConnectionUDP.Write(data)

}
