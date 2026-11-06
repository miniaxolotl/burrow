package routing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const MaxHeaders = 32 * 1024

// Exchange carries metadata only, never headers, bodies, query values or IPs.
type Exchange struct {
	Method, Path, Route     string
	StartedAt, CompletedAt  time.Time
	Status                  int
	Origin, Outcome, Reason string
}

type responseBody struct {
	io.ReadCloser
	failed bool
}

func (b *responseBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil && err != io.EOF {
		b.failed = true
	}
	return n, err
}

type Observation interface {
	Wrap(io.ReadCloser) io.ReadCloser
	Complete()
	Unavailable(string)
}

// DialLoopback does not resolve DNS, use environment proxies or accept a host
// from public input. Both address families share a five-second preparation bound.
func DialLoopback(ctx context.Context, port uint16) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var err error
	for _, ip := range []string{"127.0.0.1", "::1"} {
		var conn net.Conn
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(int(port))))
		if err == nil {
			return conn, nil
		}
	}
	return nil, err
}

func (t *Table) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e := Exchange{Method: r.Method, Path: r.URL.EscapedPath(), StartedAt: time.Now().UTC(), Origin: "client_proxy", Outcome: "rejected", Reason: "route_rejected"}
	var body *responseBody
	finished := false
	defer func() {
		e.CompletedAt = time.Now().UTC()
		if !finished || body != nil && body.failed {
			e.Outcome = "interrupted"
			e.Reason = "stream_interrupted"
		}
		if r.Context().Err() != nil {
			e.Outcome = "interrupted"
			e.Reason = "cancelled"
		}
		if t.Report != nil {
			t.Report(e)
		}
	}()
	reject := func(code int) { e.Status = code; finished = true }
	if r.Method == http.MethodConnect {
		reject(405)
		http.Error(w, "CONNECT is not supported", 405)
		return
	}
	if r.URL.IsAbs() || r.RequestURI == "*" {
		reject(400)
		http.Error(w, "Absolute request targets are not supported", 400)
		return
	}
	headerBytes := len(r.RequestURI) + len(r.Host) + len(r.Method)
	for key, values := range r.Header {
		for _, v := range values {
			headerBytes += len(key) + len(v) + 4
		}
	}
	if headerBytes > MaxHeaders {
		reject(431)
		http.Error(w, "Headers exceed limit", 431)
		return
	}
	route, path, err := t.Match(r.URL.EscapedPath())
	if err != nil {
		reject(400)
		http.Error(w, "Unsafe route path", 400)
		return
	}
	if route.Path == "" {
		reject(404)
		http.NotFound(w, r)
		return
	}
	e.Route = route.Path
	var observation Observation
	if t.Observe != nil {
		observation = t.Observe(r, route, path)
		if observation != nil {
			r.Body = observation.Wrap(r.Body)
			defer observation.Complete()
		}
	}
	dial := t.Dial
	if dial == nil {
		dial = DialLoopback
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return dial(ctx, route.Port) }, DisableKeepAlives: true, DisableCompression: true, ResponseHeaderTimeout: 30 * time.Second, MaxResponseHeaderBytes: MaxHeaders}
	defer transport.CloseIdleConnections()
	proxy := &httputil.ReverseProxy{
		Rewrite: func(req *httputil.ProxyRequest) {
			req.Out.URL.Scheme = "http"
			req.Out.URL.Host = net.JoinHostPort("127.0.0.1", strconv.Itoa(int(route.Port)))
			req.Out.URL.RawPath = path
			req.Out.URL.Path, _ = url.PathUnescape(path)
			req.Out.Host = r.Host
			if route.HostHeader != "" {
				req.Out.Host = route.HostHeader
			}
			// There is no signed forwarding-identity contract on this wire path.
			// Do not elevate caller-supplied proxy fields to trusted identities.
			for key := range req.Out.Header {
				lower := strings.ToLower(key)
				if lower == "forwarded" || lower == "x-real-ip" || strings.HasPrefix(lower, "x-forwarded-") || lower == "x-tunnel-token" || lower == "proxy-authorization" {
					req.Out.Header.Del(key)
				}
			}
		}, Transport: transport, ErrorLog: log.New(io.Discard, "", 0),
		ModifyResponse: func(response *http.Response) error {
			e.Status = response.StatusCode
			e.Origin = "application"
			e.Outcome = "completed"
			e.Reason = "response_complete"
			if t.Headers != nil {
				t.Headers(route, response.StatusCode, time.Now().UTC())
			}
			if response.StatusCode != 101 {
				body = &responseBody{ReadCloser: response.Body}
				response.Body = body
			} else {
				e.Outcome = "upgraded"
				e.Reason = "upgrade_closed_unknown"
			}
			if observation != nil && (response.StatusCode == 101 || strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream")) {
				observation.Unavailable("streaming_or_upgrade")
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			e.Status = 502
			e.Origin = "client_proxy"
			e.Outcome = "failed"
			e.Reason = "proxy_failure"
			var operation *net.OpError
			if errors.As(err, &operation) && operation.Op == "dial" {
				e.Reason = "local_dial_failed"
			}
			http.Error(w, fmt.Sprintf("Local upstream unavailable for route %s", route.Path), 502)
		},
	}
	proxy.ServeHTTP(w, r)
	finished = true
}

// Each accepted yamux stream owns exactly one HTTP request (or one negotiated
// upgrade lifetime). No public listener, arbitrary CONNECT or pipelined follow-up.
func (t *Table) ServeStream(ctx context.Context, conn net.Conn) {
	tracked := &trackedConn{Conn: conn, done: make(chan struct{})}
	listener := &streamListener{conn: tracked}
	stop := context.AfterFunc(ctx, func() { tracked.Close() })
	defer stop()
	defer tracked.Close()
	finished := make(chan struct{})
	var finish sync.Once
	done := func() { finish.Do(func() { close(finished) }) }
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { defer done(); t.ServeHTTP(w, r) }), ReadHeaderTimeout: 10 * time.Second, MaxHeaderBytes: MaxHeaders, ErrorLog: log.New(io.Discard, "", 0), BaseContext: func(net.Listener) context.Context { return ctx }, ConnState: func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			done()
		}
	}}
	server.SetKeepAlivesEnabled(false)
	server.Serve(listener)
	<-finished
}

type trackedConn struct {
	net.Conn
	once sync.Once
	done chan struct{}
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { close(c.done) })
	return err
}

type streamListener struct {
	conn     *trackedConn
	accepted bool
}

func (l *streamListener) Accept() (net.Conn, error) {
	if !l.accepted {
		l.accepted = true
		return l.conn, nil
	}
	<-l.conn.done
	return nil, net.ErrClosed
}
func (l *streamListener) Close() error   { return l.conn.Close() }
func (l *streamListener) Addr() net.Addr { return l.conn.LocalAddr() }
