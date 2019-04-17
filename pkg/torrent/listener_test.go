package torrent

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestNewListener(t *testing.T) {

	listeners := make([]*Listener, 5)
	//var err error

	var wait sync.WaitGroup

	wait.Add(5)

	for i := 0; i < 5; i++ {
		listener, err := NewListener(8090, 8099)
		assert.NoError(t, err, "can not create listener")
		assert.EqualValues(t, 8090+i, listener.Port, "unexpected port")

		go func() {
			defer wait.Done()
			err := listener.Start()
			assert.Error(t, err, "err")
		}()

		listeners[i] = listener

	}

	for i := 0; i < 5; i++ {
		listeners[i].Close()
	}

	wait.Wait()
}
