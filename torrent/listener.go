package torrent

import (
	"fmt"
	"net"
)

type Listener struct {
	port     int
	listener net.Listener

	Connections chan net.Conn
}

func NewListener(portRangeStart, portRangeEnd int) (listener *Listener, err error) {

	listener = new(Listener)

	for port := portRangeStart; port < portRangeEnd; port++ {
		listener.port = port
		listener.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", listener.port))
		if err == nil {
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

	for {

		conn, err := l.listener.Accept()
		if err != nil {
			return err
		}

		l.Connections <- conn

	}
}

func (l *Listener) Stop() {

	_ = l.listener.Close()

}
