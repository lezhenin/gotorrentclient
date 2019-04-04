package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"math/rand"
	"net"
	"sync"
	"testing"
)

func TestTracker_Run(t *testing.T) {

	infoHash := make([]byte, 20)
	myPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(myPeerId)

	interiorConn, exteriorConn := net.Pipe()

	tracker, err := NewTracker(myPeerId, infoHash, interiorConn)
	if err != nil {
		panic(err)
	}

	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		tracker.Run()
	}()

	downloaded := rand.Uint64()
	uploaded := rand.Uint64()
	left := rand.Uint64()
	port := uint16(rand.Int())
	peersCount := rand.Uint32()

	tracker.announceRequestChannel <- AnnounceRequest{
		Started,
		downloaded,
		uploaded,
		left,
		port,
		peersCount}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("Con not read 16 bytes: n = %d", n))

	trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
	actionId := binary.BigEndian.Uint32(buffer[8:12])

	assert.EqualValues(t, 0x41727101980, trackerProtocolId, "Wrong protocol ID")
	assert.EqualValues(t, 0, actionId, "Wrong action ID")

	copy(buffer[0:8], buffer[8:16])
	rand.Read(buffer[8:16])
	connectionId := binary.BigEndian.Uint64(buffer[8:16])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("Con not write 16 bytes: n = %d", n))

	buffer = make([]byte, 98)
	n, err = io.ReadAtLeast(exteriorConn, buffer, 98)
	assert.NoError(t, err, fmt.Sprintf("Con not read 98 bytes: n = %d", n))

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

	transactionId := binary.BigEndian.Uint32(buffer[12:16])

	assert.EqualValues(t, connectionId, binary.BigEndian.Uint64(buffer[0:8]), "Wrong connection id")
	assert.EqualValues(t, 1, binary.BigEndian.Uint32(buffer[8:12]), "Wrong action id")
	assert.EqualValues(t, 1, binary.BigEndian.Uint32(buffer[8:12]), "Wrong action id")
	assert.True(t, bytes.Compare(infoHash, buffer[16:36]) == 0, "Wrong info hash")
	assert.True(t, bytes.Compare(myPeerId, buffer[36:56]) == 0, "Wrong peer id")
	assert.EqualValues(t, downloaded, binary.BigEndian.Uint64(buffer[56:64]), "Wrong downloaded")
	assert.EqualValues(t, left, binary.BigEndian.Uint64(buffer[64:72]), "Wrong left")
	assert.EqualValues(t, uploaded, binary.BigEndian.Uint64(buffer[72:80]), "Wrong uploaded")
	assert.EqualValues(t, 2, binary.BigEndian.Uint32(buffer[80:84]), "Wrong event id")
	assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[84:88]), "Wrong ip address")
	assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[88:92]), "Wrong key")
	assert.EqualValues(t, peersCount, binary.BigEndian.Uint32(buffer[92:96]), "Wrong peer count")
	assert.EqualValues(t, port, binary.BigEndian.Uint16(buffer[96:98]), "Wrong port")

	//Offset      Size            Name            Value
	//0           32-bit integer  action          1 // announce
	//4           32-bit integer  transaction_id
	//8           32-bit integer  interval
	//12          32-bit integer  leechers
	//16          32-bit integer  Seeders
	//20 + 6 * n  32-bit integer  IP address
	//24 + 6 * n  16-bit integer  TCP port
	//20 + 6 * N

	interval := rand.Uint32()
	seeders := rand.Uint32()
	leechers := rand.Uint32()
	ips := []uint32{rand.Uint32(), rand.Uint32()}
	ports := []uint16{uint16(rand.Uint32()), uint16(rand.Uint32())}

	buffer = make([]byte, 32)
	binary.BigEndian.PutUint32(buffer[0:4], 1)
	binary.BigEndian.PutUint32(buffer[4:8], transactionId)
	binary.BigEndian.PutUint32(buffer[8:12], interval)
	binary.BigEndian.PutUint32(buffer[12:16], leechers)
	binary.BigEndian.PutUint32(buffer[16:20], seeders)
	binary.BigEndian.PutUint32(buffer[20:24], ips[0])
	binary.BigEndian.PutUint16(buffer[24:26], ports[0])
	binary.BigEndian.PutUint32(buffer[26:30], ips[1])
	binary.BigEndian.PutUint16(buffer[30:32], ports[1])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("Can not write 32 bytes: n = %d", n))

	response := <-tracker.announceResponseChannel

	assert.EqualValues(t, interval, response.AnnounceInterval, "Wrong announce interval")
	assert.EqualValues(t, seeders, response.SeedersCount, "Wrong seeder count")
	assert.EqualValues(t, leechers, response.LechersCount, "Wrong leecher count")
	assert.EqualValues(t, 2, len(response.Peers), "Wrong peer count")

	for i := 0; i < 2; i++ {
		ipBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(ipBytes, ips[i])
		addrString := fmt.Sprintf("%d.%d.%d.%d:%d",
			ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3], ports[i])
		assert.EqualValues(t, addrString, response.Peers[i], "Wrong peer address")
	}

	//wait.Wait()

}
