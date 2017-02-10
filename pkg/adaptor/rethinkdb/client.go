package rethinkdb

import (
	"github.com/compose/transporter/pkg/client"

	r "gopkg.in/gorethink/gorethink.v3"
)

var (
	_ client.Client = &Client{}
)

type Client struct {
	session *r.Session
}

type Session struct {
	session *r.Session
}

// Connect wraps the underlying session to the RethinkDB database
func (c *Client) Connect() (client.Session, error) {
	if c.session == nil {
		s, err := r.Connect(r.ConnectOpts{})
		if err != nil {
			return nil, err
		}
		c.session = s
	}
	return &Session{c.session}, nil
}

func (s *Session) Close() {
	s.session.Close(r.CloseOpts{NoReplyWait: false})
}
