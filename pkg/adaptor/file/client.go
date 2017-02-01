package file

import (
	"os"
	"strings"

	"github.com/compose/transporter/pkg/client"
)

var (
	_ client.Client = &Client{}
)

// ClientOptionFunc is a function that configures a Client.
// It is used in NewClient.
type ClientOptionFunc func(*Client) error

// Client represents a client to the underlying File source.
type Client struct {
	uri string

	file *os.File
}

// DefaultURI is the default file, outputs to stdout
var (
	DefaultURI = "stdout://"
)

// NewClient creates a default file client
func NewClient(options ...ClientOptionFunc) (*Client, error) {
	// Set up the client
	c := &Client{
		uri: DefaultURI,
	}

	// Run the options on it
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// WithURI defines the full connection string of the MongoDB database.
func WithURI(uri string) ClientOptionFunc {
	return func(c *Client) error {
		c.uri = uri
		return nil
	}
}

// Connect tests the mongodb connection and initializes the mongo session
func (c *Client) Connect() (client.Session, error) {
	if c.file == nil {
		if err := c.initFile(); err != nil {
			return nil, err
		}
	}
	return &Session{c.file}, nil
}

func (c *Client) initFile() error {
	if strings.HasPrefix(c.uri, "stdout://") {
		c.file = os.Stdout
		return nil
	}
	name := strings.Replace(c.uri, "file://", "", 1)
	f, err := os.OpenFile(name, os.O_RDWR, 0666)
	if os.IsNotExist(err) {
		f, err = os.Create(name)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	c.file = f
	return nil
}
