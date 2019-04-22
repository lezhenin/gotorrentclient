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
	"time"
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
		err := tracker.Run()
		assert.NoError(t, err, "tracker finished with error")
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
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
	actionId := binary.BigEndian.Uint32(buffer[8:12])

	assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
	assert.EqualValues(t, 0, actionId, "wrong action ID")

	copy(buffer[0:8], buffer[8:16])
	rand.Read(buffer[8:16])
	connectionId := binary.BigEndian.Uint64(buffer[8:16])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	buffer = make([]byte, 98)
	n, err = io.ReadAtLeast(exteriorConn, buffer, 98)
	assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

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

	assert.EqualValues(t, connectionId, binary.BigEndian.Uint64(buffer[0:8]), "wrong connection id")
	assert.EqualValues(t, 1, binary.BigEndian.Uint32(buffer[8:12]), "wrong action id")
	assert.EqualValues(t, 1, binary.BigEndian.Uint32(buffer[8:12]), "wrong action id")
	assert.True(t, bytes.Compare(infoHash, buffer[16:36]) == 0, "wrong info hash")
	assert.True(t, bytes.Compare(myPeerId, buffer[36:56]) == 0, "wrong peer id")
	assert.EqualValues(t, downloaded, binary.BigEndian.Uint64(buffer[56:64]), "wrong downloaded")
	assert.EqualValues(t, left, binary.BigEndian.Uint64(buffer[64:72]), "wrong left")
	assert.EqualValues(t, uploaded, binary.BigEndian.Uint64(buffer[72:80]), "wrong uploaded")
	assert.EqualValues(t, 2, binary.BigEndian.Uint32(buffer[80:84]), "wrong event id")
	assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[84:88]), "wrong ip address")
	assert.EqualValues(t, 0, binary.BigEndian.Uint32(buffer[88:92]), "wrong key")
	assert.EqualValues(t, peersCount, binary.BigEndian.Uint32(buffer[92:96]), "wrong peer count")
	assert.EqualValues(t, port, binary.BigEndian.Uint16(buffer[96:98]), "wrong port")

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
	assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

	response := <-tracker.announceResponseChannel

	assert.EqualValues(t, interval, response.AnnounceInterval, "wrong announce interval")
	assert.EqualValues(t, seeders, response.SeedersCount, "wrong seeder count")
	assert.EqualValues(t, leechers, response.LechersCount, "wrong leecher count")
	assert.EqualValues(t, 2, len(response.Peers), "wrong peer count")

	for i := 0; i < 2; i++ {
		ipBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(ipBytes, ips[i])
		addrString := fmt.Sprintf("%d.%d.%d.%d:%d",
			ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3], ports[i])
		assert.EqualValues(t, addrString, response.Peers[i], "wrong peer address")
	}

	tracker.Close()
	wait.Wait()

}

func TestTracker_Run_Disconnect(t *testing.T) {

	infoHash := make([]byte, 20)
	myPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(myPeerId)

	interiorConn, _ := net.Pipe()

	tracker, err := NewTracker(myPeerId, infoHash, interiorConn)
	if err != nil {
		panic(err)
	}

	var wait sync.WaitGroup
	wait.Add(1)

	go func() {
		defer wait.Done()
		err := tracker.Run()
		assert.Error(t, err, "tracker finished without error")
	}()

	time.Sleep(time.Millisecond)

	_ = interiorConn.Close()

	tracker.announceRequestChannel <- AnnounceRequest{
		Started,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	wait.Wait()
}

func TestTracker_Run_AfterClose(t *testing.T) {

	infoHash := make([]byte, 20)
	myPeerId := make([]byte, 20)

	rand.Read(infoHash)
	rand.Read(myPeerId)

	interiorConn, _ := net.Pipe()

	tracker, err := NewTracker(myPeerId, infoHash, interiorConn)
	if err != nil {
		panic(err)
	}

	var wait sync.WaitGroup
	wait.Add(1)

	go func() {
		defer wait.Done()
		err := tracker.Run()
		assert.NoError(t, err, "tracker finished with error")
	}()

	time.Sleep(time.Millisecond)

	tracker.Close()
	wait.Wait()

	err = tracker.Run()
	assert.Error(t, err, "tracker run after close")

}

func TestTracker_Run_WrongConnectionResponseAction(t *testing.T) {

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
		err := tracker.Run()
		assert.Error(t, err, "tracker finished without error")
	}()

	tracker.announceRequestChannel <- AnnounceRequest{
		Started,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	binary.BigEndian.PutUint32(buffer[0:4], 1)
	copy(buffer[4:8], buffer[12:16])
	rand.Read(buffer[8:16])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	wait.Wait()
}

func TestTracker_Run_WrongConnectionResponseTransactionId(t *testing.T) {

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
		err := tracker.Run()
		assert.Error(t, err, "tracker finished without error")
	}()

	tracker.announceRequestChannel <- AnnounceRequest{
		Started,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	copy(buffer[0:8], buffer[8:16])
	buffer[5] += 23
	rand.Read(buffer[8:16])

	n, err = exteriorConn.Write(buffer[0:16])
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	wait.Wait()
}

func TestTracker_Run_WrongConnectionResponseLength(t *testing.T) {

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
		err := tracker.Run()
		assert.Error(t, err, "tracker finished without error")
	}()

	tracker.announceRequestChannel <- AnnounceRequest{
		Started,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	n, err = exteriorConn.Write(buffer[:10])
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	wait.Wait()
}

func TestTracker_Run_WrongAnnounceResponseAction(t *testing.T) {

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
		err := tracker.Run()
		assert.Error(t, err, "tracker finished without error")
	}()

	tracker.announceRequestChannel <- AnnounceRequest{
		None,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
	actionId := binary.BigEndian.Uint32(buffer[8:12])

	assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
	assert.EqualValues(t, 0, actionId, "wrong action ID")

	copy(buffer[0:8], buffer[8:16])
	rand.Read(buffer[8:16])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	buffer = make([]byte, 98)
	n, err = io.ReadAtLeast(exteriorConn, buffer, 98)
	assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

	transactionId := binary.BigEndian.Uint32(buffer[12:16])

	buffer = make([]byte, 32)
	rand.Read(buffer)

	binary.BigEndian.PutUint32(buffer[0:4], 0)
	binary.BigEndian.PutUint32(buffer[4:8], transactionId)

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

	wait.Wait()

}

func TestTracker_Run_WrongAnnounceResponseTransactionId(t *testing.T) {

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
		err := tracker.Run()
		assert.Error(t, err, "tracker finished without error")
	}()

	tracker.announceRequestChannel <- AnnounceRequest{
		None,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
	actionId := binary.BigEndian.Uint32(buffer[8:12])

	assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
	assert.EqualValues(t, 0, actionId, "wrong action ID")

	copy(buffer[0:8], buffer[8:16])
	rand.Read(buffer[8:16])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	buffer = make([]byte, 98)
	n, err = io.ReadAtLeast(exteriorConn, buffer, 98)
	assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

	transactionId := binary.BigEndian.Uint32(buffer[12:16])

	buffer = make([]byte, 32)
	rand.Read(buffer)

	binary.BigEndian.PutUint32(buffer[0:4], 1)
	binary.BigEndian.PutUint32(buffer[4:8], transactionId+24)

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

	wait.Wait()

}

func TestTracker_Run_WrongAnnounceResponseLength(t *testing.T) {

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
		err := tracker.Run()
		assert.Error(t, err, "tracker finished without error")
	}()

	tracker.announceRequestChannel <- AnnounceRequest{
		None,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
	actionId := binary.BigEndian.Uint32(buffer[8:12])

	assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
	assert.EqualValues(t, 0, actionId, "wrong action ID")

	copy(buffer[0:8], buffer[8:16])
	rand.Read(buffer[8:16])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	buffer = make([]byte, 98)
	n, err = io.ReadAtLeast(exteriorConn, buffer, 98)
	assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

	transactionId := binary.BigEndian.Uint32(buffer[12:16])

	buffer = make([]byte, 32)
	rand.Read(buffer)

	binary.BigEndian.PutUint32(buffer[0:4], 1)
	binary.BigEndian.PutUint32(buffer[4:8], transactionId)

	n, err = exteriorConn.Write(buffer[:15])
	assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

	wait.Wait()

}

func TestTracker_Run_ConnectionExpiration(t *testing.T) {

	if testing.Short() {
		t.Skip("skip in short mode: test duration is about 1m")
	}

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
		err := tracker.Run()
		assert.NoError(t, err, "tracker finished with error")
	}()

	tracker.announceRequestChannel <- AnnounceRequest{
		None,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer := make([]byte, 16)
	n, err := io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	trackerProtocolId := binary.BigEndian.Uint64(buffer[0:8])
	actionId := binary.BigEndian.Uint32(buffer[8:12])

	assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
	assert.EqualValues(t, 0, actionId, "wrong action ID")

	copy(buffer[0:8], buffer[8:16])
	rand.Read(buffer[8:16])

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 16 bytes: n = %d", n))

	buffer = make([]byte, 98)
	n, err = io.ReadAtLeast(exteriorConn, buffer, 98)
	assert.NoError(t, err, fmt.Sprintf("can not read 98 bytes: n = %d", n))

	transactionId := binary.BigEndian.Uint32(buffer[12:16])

	buffer = make([]byte, 32)
	rand.Read(buffer)

	binary.BigEndian.PutUint32(buffer[0:4], 1)
	binary.BigEndian.PutUint32(buffer[4:8], transactionId)

	n, err = exteriorConn.Write(buffer)
	assert.NoError(t, err, fmt.Sprintf("can not write 32 bytes: n = %d", n))

	time.Sleep(time.Minute)

	tracker.announceRequestChannel <- AnnounceRequest{
		None,
		rand.Uint64(),
		rand.Uint64(),
		rand.Uint64(),
		uint16(rand.Int()),
		rand.Uint32(),
	}

	buffer = make([]byte, 16)
	n, err = io.ReadAtLeast(exteriorConn, buffer, 16)
	assert.NoError(t, err, fmt.Sprintf("can not read 16 bytes: n = %d", n))

	trackerProtocolId = binary.BigEndian.Uint64(buffer[0:8])
	actionId = binary.BigEndian.Uint32(buffer[8:12])

	assert.EqualValues(t, 0x41727101980, trackerProtocolId, "wrong protocol ID")
	assert.EqualValues(t, 0, actionId, "wrong action ID")

	tracker.Close()

	wait.Wait()

}
