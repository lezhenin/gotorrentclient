package torrent

import (
	"encoding/binary"
	"fmt"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"sync"
	"time"
)

const connectionLifetime = time.Minute

type Event uint32

const (
	None      Event = 0
	Completed Event = 1
	Started   Event = 2
	Stopped   Event = 3
)

type AnnounceRequest struct {
	Event      Event
	Downloaded uint64
	Uploaded   uint64
	Left       uint64
	Port       uint16
	PeersCount uint32
}

type AnnounceResponse struct {
	AnnounceInterval uint32
	SeedersCount     uint32
	LechersCount     uint32
	Peers            []string
}

type Tracker struct {
	connection net.Conn

	expire        bool
	connectionId  uint64
	transactionId uint32

	expirationTimer *time.Timer

	announceRequestChannel  chan AnnounceRequest
	announceResponseChannel chan AnnounceResponse

	peerId   []byte
	infoHash []byte

	closed bool

	closeMutex sync.Mutex
	closeWait  sync.WaitGroup
	closeOnce  sync.Once
}

func NewTracker(peerId, infoHash []byte, connection net.Conn) (tracker *Tracker, err error) {

	tracker = new(Tracker)

	tracker.connection = connection

	tracker.expirationTimer = time.NewTimer(0)
	<-tracker.expirationTimer.C

	tracker.expire = true

	tracker.announceRequestChannel = make(chan AnnounceRequest, 1)
	tracker.announceResponseChannel = make(chan AnnounceResponse, 1)

	tracker.infoHash = infoHash
	tracker.peerId = peerId

	return tracker, nil

}

func (t *Tracker) Run() (err error) {

	if t.closed == true {
		return errors.Annotate(
			errors.New("connection was already closed"),
			"tracker run")
	}

	trackerLogger.WithFields(logrus.Fields{
		"address": t.connection.RemoteAddr(),
	}).Info("tracker connection is serviced")

	t.closeWait.Add(1)
	defer t.closeWait.Done()

	for {

		select {

		case _, ok := <-t.expirationTimer.C:

			if !ok {
				return nil
			}

			trackerLogger.WithFields(logrus.Fields{
				"address":       t.connection.RemoteAddr(),
				"connection id": t.connectionId,
			}).Trace("connection id expired")

			t.expire = true

		case request, ok := <-t.announceRequestChannel:

			if !ok {
				return nil
			}

			response, err := t.announce(request)
			if err != nil {
				t.closeMutex.Lock()
				if t.closed {
					t.closeMutex.Unlock()
					return nil
				} else {

					trackerLogger.WithFields(logrus.Fields{
						"address": t.connection.RemoteAddr(),
					}).Error(err.Error())

					t.close()
					t.closeMutex.Unlock()
					return err
				}
			}

			t.announceResponseChannel <- response
		}
	}
}

func (t *Tracker) Close() {

	t.closeMutex.Lock()
	t.close()
	t.closeMutex.Unlock()
	t.closeWait.Wait()

}

func (t *Tracker) close() {
	t.closeOnce.Do(func() {
		t.closed = true
		_ = t.connection.Close()
		close(t.announceResponseChannel)
		close(t.announceRequestChannel)

		trackerLogger.WithFields(logrus.Fields{
			"address": t.connection.RemoteAddr(),
		}).Info("tracker connection is closed")

	})
}

func (t *Tracker) establishConnection() (connectionId uint64, err error) {

	if !t.expire {
		return t.connectionId, nil
	}

	transactionId := rand.Uint32()

	request := makeConnectionRequest(transactionId)
	_, err = t.connection.Write(request)

	if err != nil {
		return 0,
			errors.Annotate(err, "establish connection")
	}

	response := make([]byte, 16)
	n, err := t.connection.Read(response)
	t.connectionId, err = parseConnectionResponse(response[:n], transactionId)

	if err != nil {
		return 0,
			errors.NewNotValid(err, "establish connection")
	}

	t.expire = false
	t.expirationTimer.Reset(connectionLifetime)

	trackerLogger.WithFields(logrus.Fields{
		"address":       t.connection.RemoteAddr(),
		"connection id": t.connectionId,
	}).Trace("new connection id received")

	return t.connectionId, nil

}

func (t *Tracker) announce(request AnnounceRequest) (response AnnounceResponse, err error) {

	connectionId, err := t.establishConnection()
	if err != nil {
		return AnnounceResponse{},
			errors.Annotate(err, "tracker announce")
	}

	transactionId := rand.Uint32()
	data := makeAnnounceRequest(t.peerId, t.infoHash, connectionId, transactionId, request)

	_, err = t.connection.Write(data)

	if err != nil {
		return AnnounceResponse{},
			errors.Annotate(err, "tracker announce")
	}

	trackerLogger.WithFields(logrus.Fields{
		"address": t.connection.RemoteAddr(),
		"event":   request.Event,
		"port":    request.Port,
	}).Trace("announce request sent")

	data = make([]byte, 1024)
	n, err := t.connection.Read(data)

	if err != nil {
		return AnnounceResponse{},
			errors.Annotate(err, "tracker announce")
	}

	response, err = parseAnnounceResponse(data[:n], transactionId)

	if err != nil {
		return AnnounceResponse{},
			errors.NewNotValid(err, "tracker announce")
	}

	trackerLogger.WithFields(logrus.Fields{
		"address":     t.connection.RemoteAddr(),
		"interval":    response.AnnounceInterval,
		"peers_count": len(response.Peers),
		"peers":       response.Peers,
	}).Trace("announce response received")

	return response, nil

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
			errors.Annotate(
				errors.Errorf("response has unexpected length %d", len(response)),
				"parse connection response")
	}

	actionBytes := response[0:4]
	transactionIdBytes := response[4:8]
	connectionIdBytes := response[8:16]

	if binary.BigEndian.Uint32(transactionIdBytes) != expectedTransactionId {
		return 0,
			errors.Annotate(
				errors.Errorf("transaction id doesn't match expected value"),
				"parse connection response")
	}

	if binary.BigEndian.Uint32(actionBytes) != 0 {
		return 0,
			errors.Annotate(
				errors.Errorf("action is not connect"),
				"parse connection response")
	}

	return binary.BigEndian.Uint64(connectionIdBytes), nil

}

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

func makeAnnounceRequest(peerId, infoHash []byte, connectionId uint64, transactionId uint32, request AnnounceRequest) (data []byte) {

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

	binary.BigEndian.PutUint64(connectionIdBytes, connectionId)
	binary.BigEndian.PutUint32(actionBytes, 1) // anounce
	binary.BigEndian.PutUint32(transactionIdBytes, transactionId)
	binary.BigEndian.PutUint64(downloadedBytes, request.Downloaded)
	binary.BigEndian.PutUint64(leftBytes, request.Left)
	binary.BigEndian.PutUint64(uploadedBytes, request.Uploaded)
	binary.BigEndian.PutUint32(eventBytes, uint32(request.Event))
	binary.BigEndian.PutUint32(addressIpBytes, 0) // default
	binary.BigEndian.PutUint32(keyBytes, 0)       // default
	binary.BigEndian.PutUint32(numWantBytes, request.PeersCount)
	binary.BigEndian.PutUint16(portBytes, request.Port)

	copy(infoHashBytes, infoHash)
	copy(peerIdBytes, peerId)

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

func parseAnnounceResponse(data []byte, expectedTransactionId uint32) (response AnnounceResponse, err error) {

	if len(data) < 20 {
		return AnnounceResponse{},
			errors.Annotate(
				errors.Errorf("message length %d < 20, data = %v", len(data), data),
				"parse connection response")
	}

	actionBytes := data[0:4]
	transactionIdBytes := data[4:8]
	intervalBytes := data[8:12]
	lechersNumberBytes := data[12:16]
	seedersNumberBytes := data[16:20]

	if binary.BigEndian.Uint32(transactionIdBytes) != expectedTransactionId {
		return AnnounceResponse{},
			errors.Annotate(
				errors.Errorf("transaction id doesn't match expected value"),
				"parse connection response")
	}

	if binary.BigEndian.Uint32(actionBytes) != 1 {
		return AnnounceResponse{},
			errors.Annotate(
				errors.Errorf("action is not announce"),
				"parse connection response")
	}

	response.AnnounceInterval = binary.BigEndian.Uint32(intervalBytes)

	response.LechersCount = binary.BigEndian.Uint32(lechersNumberBytes)
	response.SeedersCount = binary.BigEndian.Uint32(seedersNumberBytes)

	receivedPeersCount := (len(data) - 20) / 6
	response.Peers = make([]string, receivedPeersCount)

	for i := 0; i < int(receivedPeersCount); i++ {

		addrBytes := data[20+6*i : 20+6*(i+1)]
		addrString := fmt.Sprintf("%d.%d.%d.%d:%d",
			addrBytes[0], addrBytes[1], addrBytes[2], addrBytes[3],
			binary.BigEndian.Uint16(addrBytes[4:6]))

		response.Peers[i] = addrString

	}

	return response, nil
}
