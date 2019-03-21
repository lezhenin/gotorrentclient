package torrent

import "sync"

type State struct {
	uploaded   uint64
	downloaded uint64
	left       uint64

	mutex sync.RWMutex
}

func NewState(left uint64) (s *State) {
	s = new(State)
	s.left = left
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
