package torrent

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
	"time"
)

type Event uint32

const (
	None      Event = 0
	Completed Event = 1
	Started   Event = 2
	Stopped   Event = 3
)

type DownloadStateObserver interface {
	GetDownloadedByteCount() uint64
	GetUploadedByteCount() uint64
	GetLeftByteCount() uint64
}

type Tracker struct {
	stateObserver DownloadStateObserver
	peerId        []byte
	infoHash      []byte
	listenPort    uint16

	Seeders []string

	trackerURL    *url.URL
	connectionUDP io.ReadWriter

	interval uint32

	expire       bool
	connectionId uint64

	lastEvent Event

	announceTimer   *time.Timer
	expirationTimer *time.Timer

	eventChannel chan Event
	errorChannel chan error
}

func (c *Tracker) routine() {

	for {

		select {

		case <-c.expirationTimer.C:
			c.expire = true

		case <-c.announceTimer.C:

			err := c.EstablishConnection()
			if err != nil {
				panic(err)
			}

			c.interval, c.Seeders, err = c.SendAnnounce(None)
			if err != nil {
				panic(err)
			}

			c.announceTimer.Reset(time.Duration(c.interval) * time.Second)

			c.errorChannel <- nil

		case event := <-c.eventChannel:

			err := c.EstablishConnection()
			if err != nil {
				c.errorChannel <- err
				continue
			}

			c.interval, c.Seeders, err = c.SendAnnounce(event)
			if err != nil {
				c.errorChannel <- err
				continue
			}

			c.errorChannel <- nil

			if event == Started {
				c.announceTimer.Reset(time.Duration(c.interval) * time.Second)
			} else if event == Completed || event == Stopped {
				break
			}

		}

	}
}

func (c *Tracker) Start() {

	if c.lastEvent == Started {
		panic("Start while started")
	}

	go c.routine()

	c.eventChannel <- Started

	err := <-c.errorChannel
	if err != nil {
		panic(err)
	}

	c.lastEvent = Started
}

func (c *Tracker) Stop() {

	if c.lastEvent == Stopped {
		panic("Stop while stopped")
	}

	c.expirationTimer.Stop()
	c.announceTimer.Stop()

	c.eventChannel <- Stopped
	err := <-c.errorChannel
	if err != nil {
		panic(err)
	}

	c.lastEvent = Stopped

}

func (c *Tracker) Complete() {

	if c.lastEvent == Completed {
		panic("Complete while Completed")
	}

	c.expirationTimer.Stop()
	c.announceTimer.Stop()

	c.eventChannel <- Completed
	err := <-c.errorChannel
	if err != nil {
		panic(err)
	}

	c.lastEvent = Completed

}

func NewTrackerConnection(
	announce string, peerId []byte, infoHash []byte,
	port uint16, stateObserver DownloadStateObserver) (trackerConnection *Tracker, err error) {

	trackerConnection = new(Tracker)

	if len(peerId) != 20 {
		return nil,
			fmt.Errorf("new torrent connection: peer id has wrong len %d",
				len(peerId))
	}

	trackerConnection.peerId = peerId

	if len(infoHash) != 20 {
		return nil,
			fmt.Errorf("new torrent connection: info hash has wrong len %d",
				len(infoHash))
	}

	trackerConnection.infoHash = infoHash

	trackerConnection.trackerURL, err = url.Parse(announce)
	if err != nil {
		return nil, err
	}

	if trackerConnection.trackerURL.Scheme != "udp" {
		return nil,
			fmt.Errorf("new torrent connection: %s torrent protocol is not supported yet",
				trackerConnection.trackerURL.Scheme)
	}

	trackerConnection.stateObserver = stateObserver

	trackerConnection.listenPort = port

	trackerConnection.connectionUDP, err =
		net.Dial("udp", trackerConnection.trackerURL.Host)

	if err != nil {
		panic(err)
	}

	log.Printf("UDP connection to %s is created\n", trackerConnection.trackerURL.Host)

	trackerConnection.expirationTimer = time.NewTimer(0)
	trackerConnection.announceTimer = time.NewTimer(0)

	<-trackerConnection.expirationTimer.C
	<-trackerConnection.announceTimer.C

	trackerConnection.expire = true
	trackerConnection.lastEvent = None

	trackerConnection.eventChannel = make(chan Event, 1)
	trackerConnection.errorChannel = make(chan error, 1)

	return trackerConnection, nil

}

func (c *Tracker) EstablishConnection() (err error) {

	if !c.expire {
		return nil
	}

	transactionId := rand.Uint32()

	request := makeConnectionRequest(transactionId)
	_, err = c.connectionUDP.Write(request)

	if err != nil {
		panic(err)
	}

	response := make([]byte, 16)
	_, err = c.connectionUDP.Read(response)
	c.connectionId, err = parseConnectionResponse(response, transactionId)

	if err != nil {
		panic(err)
	}

	c.expire = false
	c.expirationTimer.Reset(time.Minute)

	log.Printf("Tracker to torrent %s was established: c id = %d",
		c.trackerURL.String(), c.connectionId)

	return nil

}

func (c *Tracker) SendAnnounce(event Event) (interval uint32, seeders []string, err error) {

	transactionId := rand.Uint32()
	request := c.makeAnnounceRequest(transactionId, event)

	_, err = c.connectionUDP.Write(request)

	if err != nil {
		panic(err)
	}

	response := make([]byte, 512)
	n, err := c.connectionUDP.Read(response)

	if err != nil {
		panic(err)
	}

	response = response[0:n]
	interval, seeders, err = parseAnnounceResponse(response, transactionId)

	if err != nil {
		panic(err)
	}

	log.Printf("Announce with event %d is send: %d peers received, interval %d.\n",
		event, len(seeders), interval)

	return interval, seeders, nil

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
			fmt.Errorf("parse connection response"+
				" response has unexpected length %d",
				len(response))
	}

	actionBytes := response[0:4]
	transactionIdBytes := response[4:8]
	connectionIdBytes := response[8:16]

	if binary.BigEndian.Uint32(transactionIdBytes) != expectedTransactionId {
		return 0,
			fmt.Errorf("parse connection response:" +
				" transaction id doesn't match expected value")
	}

	if binary.BigEndian.Uint32(actionBytes) != 0 {
		return 0,
			fmt.Errorf("parse connection response: action != 0")
	}

	return binary.BigEndian.Uint64(connectionIdBytes), nil

}

//func (torrent *Tracker) GetSeeders (n int32) {
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
//96      16-bit integer  listenPort
//98

func (c *Tracker) makeAnnounceRequest(transactionId uint32, event Event) (data []byte) {

	data = make([]byte, 98)

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

	binary.BigEndian.PutUint64(connectionIdBytes, c.connectionId)
	binary.BigEndian.PutUint32(actionBytes, 1)
	binary.BigEndian.PutUint32(transactionIdBytes, transactionId)
	binary.BigEndian.PutUint64(downloadedBytes, c.stateObserver.GetDownloadedByteCount())
	binary.BigEndian.PutUint64(leftBytes, c.stateObserver.GetLeftByteCount())
	binary.BigEndian.PutUint64(uploadedBytes, c.stateObserver.GetUploadedByteCount())
	binary.BigEndian.PutUint32(eventBytes, uint32(event))
	binary.BigEndian.PutUint32(addressIpBytes, 0) // default
	binary.BigEndian.PutUint32(keyBytes, 0)
	binary.BigEndian.PutUint32(numWantBytes, 50) // default
	binary.BigEndian.PutUint16(portBytes, 6881)

	copy(infoHashBytes, c.infoHash)
	copy(peerIdBytes, c.peerId)

	return data
}

//Offset      Size            Name            Value
//0           32-bit integer  action          1 // announce
//4           32-bit integer  transaction_id
//8           32-bit integer  interval
//12          32-bit integer  leechers
//16          32-bit integer  Seeders
//20 + 6 * n  32-bit integer  IP address
//24 + 6 * n  16-bit integer  TCP port
//20 + 6 * N

func parseAnnounceResponse(response []byte, expectedTransactionId uint32) (
	interval uint32, seederAddresses []string, err error) {

	if len(response) < 20 {
		return 0, nil,
			fmt.Errorf("parse announce response:"+
				" message length %d < 20, data = %v", len(response), response)
	}

	actionBytes := response[0:4]
	transactionIdBytes := response[4:8]
	intervalBytes := response[8:12]
	//leechersNumberBytes := response[12:16]
	//seedersNumberBytes := response[16:20]

	if binary.BigEndian.Uint32(transactionIdBytes) != expectedTransactionId {
		return 0, nil,
			fmt.Errorf("parse announce response:" +
				" transaction id doesn't match expected value")
	}

	if binary.BigEndian.Uint32(actionBytes) != 1 {
		return 0, nil,
			fmt.Errorf("parse announce response: action != 1")
	}

	//leechersNumber := binary.BigEndian.Uint32(leechersNumberBytes)
	//seedersNumber := binary.BigEndian.Uint32(seedersNumberBytes)

	receivedPeersCount := (len(response) - 20) / 6
	seederAddresses = make([]string, receivedPeersCount)

	for i := 0; i < int(receivedPeersCount); i++ {

		addrBytes := response[20+6*i : 20+6*(i+1)]
		addrString := fmt.Sprintf("%d.%d.%d.%d:%d",
			addrBytes[0], addrBytes[1], addrBytes[2], addrBytes[3],
			binary.BigEndian.Uint16(addrBytes[4:6]))

		fmt.Println(addrString)

		seederAddresses[i] = addrString
		if err != nil {
			return 0, nil, err
		}

	}

	return binary.BigEndian.Uint32(intervalBytes), seederAddresses, nil
}
