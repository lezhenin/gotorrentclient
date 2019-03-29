package torrent

import (
	"fmt"
	"net"
	"sync"
)

type Listener struct {
	Port        int
	Connections chan net.Conn

	listener net.Listener

	wait sync.WaitGroup
}

func NewListener(portRangeStart, portRangeEnd int) (listener *Listener, err error) {

	listener = new(Listener)

	for port := portRangeStart; port < portRangeEnd; port++ {
		listener.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Port = port
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("new listener: probably ports from %d to %d unavailable",
			portRangeStart, portRangeEnd)
	}

	listener.Connections = make(chan net.Conn)

	return listener, nil
}

func (l *Listener) Start() (err error) {

	l.wait.Add(1)
	defer l.wait.Done()

	for {

		conn, err := l.listener.Accept()
		if err != nil {
			return err
		}

		l.Connections <- conn

	}
}

func (l *Listener) Close() {

	_ = l.listener.Close()
	l.wait.Wait()
}
