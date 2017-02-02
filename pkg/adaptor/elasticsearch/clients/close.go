package clients

import (
	"sync"

	"github.com/compose/transporter/pkg/client"
	"github.com/compose/transporter/pkg/log"
)

// Close takes care of receiving on the done channel and then properly cleaning up
// the session and WaitGroup.
func Close(done chan struct{}, wg *sync.WaitGroup, s client.Session) {
	for {
		select {
		case <-done:
			log.Debugln("received done channel")
			if c, ok := s.(client.Closeable); ok {
				c.Close()
			}
			wg.Done()
			return
		}
	}
}
