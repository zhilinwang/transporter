package v2

import (
	"fmt"
	"sync"
	"time"

	elastic "gopkg.in/olivere/elastic.v3"

	"github.com/compose/transporter/pkg/adaptor/elasticsearch/clients"
	"github.com/compose/transporter/pkg/client"
	"github.com/compose/transporter/pkg/log"
	"github.com/compose/transporter/pkg/message"
	"github.com/compose/transporter/pkg/message/ops"
	version "github.com/hashicorp/go-version"
)

var (
	_ client.Writer  = &Writer{}
	_ client.Session = &Writer{}
)

// Writer implements client.Writer and client.Session for sending requests to an elasticsearch
// cluster via its _bulk API.
type Writer struct {
	bp     *elastic.BulkProcessor
	logger log.Logger
}

func init() {
	constraint, _ := version.NewConstraint(">= 2.0, < 5.0")
	clients.Add("v2", constraint, func(done chan struct{}, wg *sync.WaitGroup, opts *clients.ClientOptions) (client.Writer, error) {
		esOptions := []elastic.ClientOptionFunc{
			elastic.SetURL(opts.URLs...),
			elastic.SetSniff(false),
			elastic.SetHttpClient(opts.HTTPClient),
			elastic.SetMaxRetries(2),
		}
		if opts.UserInfo != nil {
			if pwd, ok := opts.UserInfo.Password(); ok {
				esOptions = append(esOptions, elastic.SetBasicAuth(opts.UserInfo.Username(), pwd))
			}
		}
		esClient, err := elastic.NewClient(esOptions...)
		if err != nil {
			return nil, err
		}
		w := &Writer{
			logger: log.With("writer", "elasticsearch").With("version", 2).With("path", opts.Path),
		}
		p, err := esClient.BulkProcessor().
			Name("TransporterWorker-1").
			Workers(2).
			BulkActions(1000).               // commit if # requests >= 1000
			BulkSize(2 << 20).               // commit if size of requests >= 2 MB
			FlushInterval(30 * time.Second). // commit every 30s
			After(w.postBulkProcessor).
			Do()
		if err != nil {
			return nil, err
		}
		w.bp = p
		wg.Add(1)
		go clients.Close(done, wg, w)
		return w, nil
	})
}

func (w *Writer) Write(msg message.Msg) func(client.Session) error {
	return func(s client.Session) error {
		i, t, _ := message.SplitNamespace(msg)
		var id string
		if _, ok := msg.Data()["_id"]; ok {
			id = msg.ID()
			msg.Data().Delete("_id")
		}

		var br elastic.BulkableRequest
		switch msg.OP() {
		case ops.Delete:
			// we need to flush any pending writes here or this could fail because we're using
			// more than 1 worker
			w.bp.Flush()
			br = elastic.NewBulkDeleteRequest().Index(i).Type(t).Id(id)
		case ops.Insert:
			br = elastic.NewBulkIndexRequest().Index(i).Type(t).Id(id).Doc(msg.Data())
		case ops.Update:
			br = elastic.NewBulkUpdateRequest().Index(i).Type(t).Id(id).Doc(msg.Data())
		}
		w.bp.Add(br)
		return nil
	}
}

// Close is called by clients.Close() when it receives on the done channel.
func (w *Writer) Close() {
	w.logger.Infoln("closing BulkProcessor")
	w.bp.Close()
}

func (w *Writer) postBulkProcessor(executionID int64, reqs []elastic.BulkableRequest, resp *elastic.BulkResponse, err error) {
	ctxLog := w.logger.
		With("executionID", executionID).
		With("took", fmt.Sprintf("%dms", resp.Took)).
		With("succeeeded", len(resp.Succeeded()))
	if err != nil {
		ctxLog.With("failed", len(resp.Failed())).Errorln(err)
		return
	}
	ctxLog.Infoln("_bulk flush completed")
}
