package file

import (
	"fmt"
	"regexp"

	"github.com/compose/transporter/pkg/adaptor"
	"github.com/compose/transporter/pkg/client"
	"github.com/compose/transporter/pkg/log"
	"github.com/compose/transporter/pkg/message"
	"github.com/compose/transporter/pkg/pipe"
)

// File is an adaptor that can be used as a
// source / sink for file's on disk, as well as a sink to stdout.
type File struct {
	uri         string
	pipe        *pipe.Pipe
	path        string
	client      client.Client
	writer      client.Writer
	reader      client.Reader
	doneChannel chan struct{}
}

// Description for file adaptor
func (f *File) Description() string {
	return "an adaptor that reads / writes files"
}

const sampleConfig = `
- stdout:
    type: file
    uri: stdout://
`

// SampleConfig for file adaptor
func (f *File) SampleConfig() string {
	return sampleConfig
}

func init() {
	adaptor.Add("file", func(p *pipe.Pipe, path string, extra adaptor.Config) (adaptor.Adaptor, error) {
		var (
			conf Config
			err  error
		)
		if err = extra.Construct(&conf); err != nil {
			return nil, adaptor.NewError(adaptor.CRITICAL, path, fmt.Sprintf("Can't configure adaptor (%s)", err.Error()), nil)
		}

		f := &File{
			uri:         conf.URI,
			pipe:        p,
			path:        path,
			writer:      newWriter(),
			reader:      newReader(),
			doneChannel: make(chan struct{}),
		}
		clientOptions := []ClientOptionFunc{
			WithURI(conf.URI),
		}

		f.client, err = NewClient(clientOptions...)
		return f, err
	})
}

// Start the file adaptor
// TODO: we only know how to listen on stdout for now
func (f *File) Start() (err error) {
	log.With("file", f.uri).Infoln("adaptor Starting...")
	defer func() {
		f.pipe.Stop()
	}()

	s, err := f.client.Connect()
	if err != nil {
		return err
	}
	allFilter := func(string) bool { return true }
	readFunc := f.reader.Read(allFilter)
	msgChan, err := readFunc(s, f.doneChannel)
	if err != nil {
		return err
	}
	for msg := range msgChan {
		f.pipe.Send(msg)
	}

	log.With("file", f.uri).Infoln("adaptor Start finished...")
	return nil
}

var (
	_ adaptor.Adaptor = &File{}
)

// Listen starts the listener
func (f *File) Listen() error {
	log.With("file", f.uri).Infoln("adaptor Listening...")
	defer func() {
		log.With("file", f.uri).Infoln("adaptor Listen closing...")
		f.pipe.Stop()
	}()
	return f.pipe.Listen(f.applyOp, regexp.MustCompile(`.*`))
}

func (f *File) applyOp(msg message.Msg) (message.Msg, error) {
	_, msgColl, _ := message.SplitNamespace(msg)
	err := client.Write(f.client, f.writer, message.From(msg.OP(), msgColl, msg.Data()))
	if err != nil {
		f.pipe.Err <- adaptor.NewError(adaptor.ERROR, f.path, fmt.Sprintf("write message error (%s)", err), msg.Data)
	}
	return msg, err
}

// Stop the adaptor
func (f *File) Stop() error {
	f.pipe.Stop()
	return nil
}

// Config is used to configure the File Adaptor
type Config struct {
	// URI pointing to the resource.  We only recognize file:// and stdout:// currently
	URI string `json:"uri" doc:"the uri to connect to, ie stdout://, file:///tmp/output"`
}
