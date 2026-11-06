package capture

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"burrow/burrowctl/internal/routing"
)

type CaptureEvent struct{ Type, ID, Code string }
type captureWork struct {
	record Record
	epoch  int64
}
type Recorder struct {
	store                *Store
	tunnel, digest, path string
	epoch                atomic.Int64
	excludes             []string
	emit                 func(CaptureEvent)
	slots                chan struct{}
	queue                chan captureWork
	closed               atomic.Bool
}

func NewRecorder(store *Store, tunnel string, table *routing.Table, path string, excludes []string, emit func(CaptureEvent)) (*Recorder, error) {
	epoch, err := store.Epoch(tunnel)
	if err != nil {
		return nil, err
	}
	r := &Recorder{store: store, tunnel: tunnel, digest: Digest(table), path: path, excludes: append([]string(nil), excludes...), emit: emit, slots: make(chan struct{}, 8), queue: make(chan captureWork, 8)}
	r.epoch.Store(epoch)
	return r, nil
}
func (r *Recorder) event(kind, id, code string) {
	if r.emit != nil {
		r.emit(CaptureEvent{kind, id, code})
	}
}
func (r *Recorder) Run(ctx context.Context) {
	write := func(work captureWork) {
		defer func() { <-r.slots }()
		record := work.record
		if err := r.store.Put(record, work.epoch); err != nil {
			r.event("capture_unavailable", record.ID, "storage_or_quota")
		} else {
			r.event("capture_saved", record.ID, "")
		}
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	cleanup := time.NewTicker(time.Minute)
	defer cleanup.Stop()
	for {
		select {
		case work := <-r.queue:
			write(work)
		case <-ticker.C:
			var epoch int64
			if err := r.store.db.QueryRow("SELECT generation FROM epochs WHERE tunnel=?", Hash(r.tunnel)).Scan(&epoch); err == nil {
				r.epoch.Store(epoch)
			}
		case <-cleanup.C:
			if err := r.store.Cleanup(); err != nil {
				r.event("capture_unavailable", "", "cleanup_unavailable")
			}
		case <-ctx.Done():
			r.closed.Store(true)
			for {
				select {
				case work := <-r.queue:
					write(work)
				default:
					return
				}
			}
		}
	}
}
func (r *Recorder) Begin(req *http.Request, route routing.Route, _ string) routing.Observation {
	o := &observation{recorder: r, record: Record{ID: rand.Text(), TunnelID: r.tunnel, Method: req.Method, Path: req.URL.EscapedPath(), Host: req.Host, Route: route, Headers: req.Header.Clone(), ConfigDigest: r.digest, ConfigPath: r.path, CreatedAt: time.Now().UTC()}, expected: req.ContentLength, epoch: r.epoch.Load()}
	for _, prefix := range r.excludes {
		if o.record.Path == prefix || strings.HasPrefix(o.record.Path, strings.TrimSuffix(prefix, "/")+"/") {
			o.reason = "excluded"
		}
	}
	if req.Method == "CONNECT" || req.Header.Get("Upgrade") != "" || strings.Contains(strings.ToLower(req.Header.Get("Accept")), "text/event-stream") || len(req.Trailer) > 0 {
		o.reason = "streaming_or_upgrade"
	}
	encoding := req.Header.Get("Content-Encoding")
	if encoding != "" && !strings.EqualFold(encoding, "identity") {
		o.reason = "content_encoding"
	}
	if req.ContentLength > MaxBody {
		o.reason = "body_limit"
	}
	if r.closed.Load() {
		o.reason = "capture_stopped"
	}
	if o.reason == "" {
		select {
		case r.slots <- struct{}{}:
			o.owns = true
		default:
			o.reason = "writer_backpressure"
		}
	}
	if o.reason != "" {
		return o
	}
	o.record.Redacted = stripHeaders(o.record.Headers)
	query, redacted, err := redactQuery(req.URL.RawQuery)
	if err != nil {
		o.reason = "invalid_query"
	} else {
		o.record.RawQuery = query
		o.record.Redacted = o.record.Redacted || redacted
	}
	if req.Body == nil || req.Body == http.NoBody {
		o.complete = true
	}
	return o
}

type observation struct {
	mu                       sync.Mutex
	recorder                 *Recorder
	record                   Record
	expected                 int64
	read                     int64
	epoch                    int64
	complete, finished, owns bool
	reason                   string
}
type teeBody struct {
	io.ReadCloser
	observation *observation
}

func (o *observation) Wrap(body io.ReadCloser) io.ReadCloser {
	if body == nil {
		body = http.NoBody
	}
	return &teeBody{body, o}
}
func (b *teeBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	o := b.observation
	o.mu.Lock()
	defer o.mu.Unlock()
	o.read += int64(n)
	if o.reason == "" && o.owns && !o.finished {
		if len(o.record.Body)+n > MaxBody {
			o.reason = "body_limit"
			o.record.Body = nil
		} else {
			o.record.Body = append(o.record.Body, p[:n]...)
		}
		if err == io.EOF || o.expected >= 0 && o.read == o.expected {
			o.complete = true
		} else if err != nil {
			o.reason = "body_read_failed"
		}
	}
	return n, err
}
func (o *observation) Unavailable(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.reason = reason
	o.record.Body = nil
}
func (o *observation) Complete() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.finished {
		return
	}
	o.finished = true
	if !o.complete && o.reason == "" {
		o.reason = "incomplete"
	}
	if o.recorder.closed.Load() && o.reason == "" {
		o.reason = "capture_stopped"
	}
	if o.reason == "" {
		select {
		case o.recorder.queue <- captureWork{o.record, o.epoch}:
			return
		default:
			o.reason = "writer_backpressure"
		}
	}
	if o.owns {
		<-o.recorder.slots
	}
	o.record.Body = nil
	o.recorder.event("capture_unavailable", o.record.ID, o.reason)
}
