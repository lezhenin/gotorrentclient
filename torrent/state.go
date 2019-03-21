package torrent

import (
	"github.com/lezhenin/gotorrentclient/bitfield"
	"sync"
)

type State struct {
	uploaded   uint64
	downloaded uint64
	left       uint64
	bitfield   *bitfield.Bitfield
	finished   bool
	stopped    bool

	mutex sync.RWMutex
}

func NewState(left uint64, bitfieldLength uint) (s *State) {
	s = new(State)
	s.left = left
	s.bitfield = bitfield.NewBitfield(bitfieldLength)
	return s
}

func (s *State) Downloaded() uint64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.downloaded
}

func (s *State) Uploaded() uint64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.uploaded
}

func (s *State) Left() uint64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.left
}

func (s *State) BitfieldBytes() []byte {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.bitfield.Bytes()
}

func (s *State) Finished() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.finished
}

func (s *State) Stopped() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.stopped
}

func (s *State) IncrementDownloaded(n uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.downloaded += n
}

func (s *State) IncrementUploaded(n uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.uploaded += n
}

func (s *State) DecrementLeft(n uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.left -= n
}

func (s *State) SetBitfieldBit(index uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.bitfield.Set(index)
}

func (s *State) SetStopped(value bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stopped = value
}

func (s *State) SetFinished(value bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.finished = value
}
