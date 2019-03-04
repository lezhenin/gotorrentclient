package tracker

import (
	"encoding/binary"
	"fmt"
	"github.com/lezhenin/gotorrentclient/download"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
)

type UnexpectedLengthError struct {
	ActualLength   int
	ExpectedLength int
	MessageType    string
}

func (e UnexpectedLengthError) Error() string {
	return fmt.Sprintf("%s has unexpected length %d instead %d",
		e.MessageType, e.ActualLength, e.ExpectedLength)
}

func makeConnectionRequest(transactionId uint32) []byte {

	request := make([]byte, 16)

	protocolIdBytes := request[0:8]
	actionBytes := request[8:12]
	transactionIdBytes := request[12:16]

	binary.BigEndian.PutUint64(protocolIdBytes, 0x41727101980)
	binary.BigEndian.PutUint32(actionBytes, 0)
	binary.BigEndian.PutUint32(transactionIdBytes, transactionId)

	return request

}

func parseConnectionResponse(response []byte, expectedTransactionId uint32) (connectionId uint64, err error) {

	if len(response) != 16 {
		return 0,
			fmt.Errorf("parse connection response: response has unexpected length %d", len(response))
	}

	actionBytes := response[0:4]
	transactionIdBytes := response[4:8]
	connectionIdBytes := response[8:16]

	if binary.BigEndian.Uint32(transactionIdBytes) != expectedTransactionId {
		return 0,
			fmt.Errorf("parse connection response: transaction id doesn't match expected value")
	}

	if binary.BigEndian.Uint32(actionBytes) != 0 {
		return 0,
			fmt.Errorf("parse connection response: action is not zero")
	}

	return binary.BigEndian.Uint64(connectionIdBytes), nil

}

type TrackerConnection struct {
	DownloadState *download.State
	trackerURL    *url.URL
	ConnectionUDP io.ReadWriter
	ConnectionId  uint64
	Established   bool
}

func NewTrackerConnection(state *download.State, announce string) (trackerConnection *TrackerConnection, err error) {

	trackerConnection = new(TrackerConnection)
	trackerConnection.trackerURL, err = url.Parse(announce)
	if err != nil {
		return nil, err
	}

	if trackerConnection.trackerURL.Scheme != "udp" {
		return nil,
			fmt.Errorf("new tracker connection: %s tracker protocol is not supported yet",
				trackerConnection.trackerURL.Scheme)
	}

	return trackerConnection, nil

}

func (connection *TrackerConnection) EstablishConnection() (err error) {

	transactionId := rand.Uint32()

	fmt.Println(transactionId)

	connection.ConnectionUDP, err = net.Dial("udp", connection.trackerURL.Path)

	if err != nil {
		panic(err)
	}

	request := makeConnectionRequest(transactionId)
	_, err = connection.ConnectionUDP.Write(request)

	if err != nil {
		panic(err)
	}

	response := make([]byte, 16)
	_, err = connection.ConnectionUDP.Read(response)
	connection.ConnectionId, err = parseConnectionResponse(response, transactionId)

	if err != nil {
		panic(err)
	}

	connection.Established = true

	log.Printf("Connection to tracker %s was established: connection id = %d",
		connection.trackerURL.String(), connection.ConnectionId)

	return nil

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

func (connection *TrackerConnection) writeAnnounceRequest(transactionId uint32) (int, error) {

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
