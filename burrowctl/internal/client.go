package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"burrow/burrowctl/internal/capture"
	"burrow/burrowctl/internal/routing"
	"burrow/protocol"
	"burrow/protocol/telemetry"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

type Client struct {
	server         string
	token          string
	domain         string
	secure         bool
	tunnels        map[string]*TunnelConn
	mu             sync.RWMutex
	observer       telemetry.Observer
	requestContext context.Context
	ctx            context.Context
	cancel         context.CancelFunc
	closed         bool
	workers        sync.WaitGroup
	closeOnce      sync.Once
	closeDone      chan struct{}
	onEvent        func(LifecycleEvent)
	attachment     *Attachment
	routes         *routing.Table
	streams        chan struct{}
	capture        *CaptureSettings
	reachability   HealthObservation
	reachOnce      sync.Once
}

// LifecycleEvent reports observed transport state, never application readiness.
// Error codes are bounded vocabulary, not remote response bodies or credentials.
type LifecycleEvent struct {
	ObservedAt            time.Time
	Type                  string
	TunnelID              string
	Port                  uint16
	URL                   string
	Attempt               int
	RetryDelayMS          int64
	Code                  string
	Generation            string
	RegistrationConfirmed bool
	CaptureID             string
	RemainingMS           int64
	LifecycleConfirmed    bool
}

func (c *Client) SetEventHandler(handler func(LifecycleEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onEvent = handler
}

func (c *Client) emit(event LifecycleEvent) {
	event.ObservedAt = time.Now().UTC()
	c.mu.RLock()
	handler := c.onEvent
	c.mu.RUnlock()
	if handler != nil {
		handler(event)
	}
}

var httpClient = NewManagementHTTPClient()

type connectionRejected struct {
	status  int
	message string
}

func (e *connectionRejected) Error() string { return e.message }

type TunnelConn struct {
	ID              string
	Port            uint16
	URL             string
	wsConn          *websocket.Conn
	session         *yamux.Session
	ctx             context.Context
	cancel          context.CancelFunc
	latency         time.Duration
	reconnects      int
	totalSize       int64
	forwarded       sync.Once
	status          string
	generation      string
	dispatcher      *routing.Table
	lifecycle       *yamux.Stream
	lifecycleReader *bufio.Reader
	localHealth     map[uint16]HealthObservation
}

func NewClient(server, token, domain string, secure bool, observers ...telemetry.Observer) *Client {
	return NewClientWithContext(context.Background(), server, token, domain, secure, observers...)
}

func NewClientWithContext(ctx context.Context, server, token, domain string, secure bool, observers ...telemetry.Observer) *Client {
	var observer telemetry.Observer = (*telemetry.Reporter)(nil)
	if len(observers) > 0 && observers[0] != nil {
		observer = observers[0]
	}
	clientContext, cancel := context.WithCancel(ctx)
	return &Client{
		server:         server,
		token:          token,
		domain:         domain,
		secure:         secure,
		tunnels:        make(map[string]*TunnelConn),
		observer:       observer,
		requestContext: ctx,
		ctx:            clientContext,
		cancel:         cancel,
		closeDone:      make(chan struct{}),
		streams:        make(chan struct{}, 128),
	}
}

func (c *Client) CreateTunnel(ctx context.Context, port uint16) (string, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return "", errors.New("client has closed")
	}
	c.workers.Add(1)
	c.mu.Unlock()
	defer c.workers.Done()
	tctx, cancel := context.WithCancel(ctx)
	stopClientCancel := context.AfterFunc(c.ctx, cancel)
	connected := false
	defer func() {
		if !connected {
			cancel()
			stopClientCancel()
		}
	}()
	started := time.Now()
	outcome := telemetry.Unavailable
	defer func() { c.observer.Operation(telemetry.Connect, started, outcome) }()
	tunnelID := protocol.RandomTunnelID()
	if c.attachment != nil {
		tunnelID = c.attachment.TunnelID
	}
	dispatcher := c.routes
	if c.capture != nil {
		if dispatcher == nil {
			dispatcher = &routing.Table{Version: 1, Routes: []routing.Route{{Path: "/", Port: port}}}
		} else {
			copy := *dispatcher
			dispatcher = &copy
		}
		recorder, err := capture.NewRecorder(c.capture.Store, tunnelID, dispatcher, c.capture.ConfigPath, c.capture.Excludes, func(e capture.CaptureEvent) {
			c.emit(LifecycleEvent{Type: e.Type, TunnelID: tunnelID, Port: port, CaptureID: e.ID, Code: e.Code})
		})
		if err != nil {
			return "", err
		}
		dispatcher.Observe = recorder.Begin
		c.workers.Go(func() { recorder.Run(tctx) })
	}
	c.emit(LifecycleEvent{Type: "connecting", TunnelID: tunnelID, Port: port})
	startup, stopStartup := context.WithTimeout(tctx, 30*time.Second)
	var tc *TunnelConn
	var err error
	for {
		tc, err = c.createTunnelSession(startup, tunnelID, port)
		var rejected *connectionRejected
		if !errors.As(err, &rejected) || rejected.status != http.StatusLocked {
			break
		}
		select {
		case <-startup.Done():
			err = startup.Err()
		case <-time.After(250 * time.Millisecond):
			continue
		}
		break
	}
	stopStartup()
	if err != nil {
		c.emit(LifecycleEvent{Type: "failed", TunnelID: tunnelID, Port: port, Code: "connection_failed"})
		return "", err
	}

	tc.ctx = tctx
	tc.cancel = cancel
	tc.dispatcher = dispatcher

	c.mu.Lock()
	if c.closed || tctx.Err() != nil {
		c.mu.Unlock()
		tc.wsConn.Close()
		tc.session.Close()
		return "", context.Canceled
	}
	c.tunnels[tc.ID] = tc
	c.workers.Go(func() {
		defer stopClientCancel()
		<-tc.ctx.Done()
		c.mu.RLock()
		conn, session := tc.wsConn, tc.session
		c.mu.RUnlock()
		conn.Close()
		session.Close()
	})
	c.mu.Unlock()

	tunnelURL := tc.URL
	c.reachOnce.Do(func() { c.workers.Go(c.trackReachability) })
	c.workers.Go(func() { c.handleTunnel(tc, port) })
	connected = true

	outcome = telemetry.Success
	c.observer.Track(telemetry.TunnelOpened)
	return tunnelURL, nil
}

func (c *Client) SetRoutes(routes *routing.Table) { c.routes = routes }

type CaptureSettings struct {
	Store      *capture.Store
	ConfigPath string
	Excludes   []string
}

func (c *Client) SetCapture(settings *CaptureSettings) { c.capture = settings }
func (c *Client) CaptureStore() *capture.Store {
	if c.capture == nil {
		return nil
	}
	return c.capture.Store
}
func (c *Client) FixedConfiguration() bool {
	return c.capture != nil || c.attachment != nil || c.routes != nil
}
func (c *Client) RunReplay(id string) (capture.Job, error) {
	c.mu.Lock()
	if c.closed || c.capture == nil {
		c.mu.Unlock()
		return capture.Job{}, errors.New("capture worker unavailable")
	}
	c.workers.Add(1)
	c.mu.Unlock()
	defer c.workers.Done()
	return c.capture.Store.Run(c.ctx, id, nil)
}
func (c *Client) Routes() []routing.Route {
	if c.routes == nil {
		return nil
	}
	return append([]routing.Route(nil), c.routes.Routes...)
}

func (c *Client) createTunnelSession(ctx context.Context, tunnelID string, port uint16) (*TunnelConn, error) {
	header := http.Header{"X-Tunnel-Token": {c.token}, protocol.RegistrationHeader: {protocol.RegistrationVersion}, protocol.LifecycleHeader: {"1"}}
	if c.attachment != nil {
		id, expected, nonce, err := c.attachment.Prepare()
		if err != nil {
			return nil, err
		}
		header.Set(protocol.AttachmentHeader, "1")
		header.Set(protocol.AttachmentIDHeader, id)
		header.Set(protocol.ExpectedGenerationHeader, expected)
		header.Set(protocol.AttemptHeader, nonce)
	}
	scheme := "ws"
	if c.secure {
		scheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/tunnels/connect?tunnel_id=%s&port=%d", scheme, c.server, tunnelID, port)

	start := time.Now()
	// Gorilla sets a deadline, but cancellation after TCP dial does not itself
	// interrupt its blocking HTTP upgrade read. Close only during the handshake;
	// the tunnel's lifetime watcher owns cancellation once it has connected.
	connectContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	dialer := *websocket.DefaultDialer
	var stopHandshake func() bool
	dialer.NetDialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		connection, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err == nil {
			// Gorilla cancels its own derived dial context even on success. Watch
			// our outer handshake context so that internal cleanup cannot close an
			// already accepted transport before we stop this cancellation hook.
			stopHandshake = context.AfterFunc(connectContext, func() { connection.Close() })
		}
		return connection, err
	}
	conn, resp, err := dialer.DialContext(connectContext, wsURL, header)
	if stopHandshake != nil {
		stopHandshake()
	}
	if err != nil {
		if connectContext.Err() != nil {
			return nil, connectContext.Err()
		}
		if resp != nil {
			if c.attachment != nil && c.attachment.Reject != nil && (resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 409 || resp.StatusCode == 410) {
				if err := c.attachment.Reject(resp.StatusCode); err != nil {
					resp.Body.Close()
					return nil, err
				}
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
			resp.Body.Close()
			msg := strings.TrimSpace(string(body))
			if msg != "" {
				return nil, &connectionRejected{resp.StatusCode, fmt.Sprintf("server rejected connection: %s: %s", resp.Status, msg)}
			}
			return nil, &connectionRejected{resp.StatusCode, fmt.Sprintf("server rejected connection: %s", resp.Status)}
		}
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	conn.SetReadLimit(protocol.MaxMessageSize)
	latency := time.Since(start)

	tunnelURL := ""
	if resp != nil {
		tunnelURL = resp.Header.Get("X-Tunnel-URL")
	}
	if tunnelURL == "" {
		tunnelURL = fmt.Sprintf("https://%s.%s", tunnelID, c.domain)
	}

	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 30 * time.Second
	if c.attachment != nil {
		cfg.KeepAliveInterval = 10 * time.Second
		cfg.ConnectionWriteTimeout = 20 * time.Second
	}
	// Expected disconnects and remote closure must not corrupt the TUI output.
	cfg.LogOutput = io.Discard
	wrappedConn := &protocol.WsConn{Conn: conn}
	session, err := yamux.Client(wrappedConn, cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create yamux session: %w", err)
	}
	c.emit(LifecycleEvent{Type: "connected", TunnelID: tunnelID, Port: port, URL: tunnelURL})
	generation := ""
	var lifecycle *yamux.Stream
	var lifecycleReader *bufio.Reader
	if c.attachment != nil && resp.Header.Get(protocol.RegistrationHeader) != "1" {
		session.Close()
		conn.Close()
		return nil, errors.New("required registration capability unavailable")
	}
	if resp.Header.Get(protocol.RegistrationHeader) != "" {
		if resp.Header.Get(protocol.RegistrationHeader) != protocol.RegistrationVersion {
			session.Close()
			conn.Close()
			return nil, errors.New("unsupported registration version")
		}
		stop := context.AfterFunc(connectContext, func() { session.Close(); conn.Close() })
		control, err := session.AcceptStream()
		if err == nil {
			deadline, _ := connectContext.Deadline()
			control.SetReadDeadline(deadline)
			var ack protocol.Registration
			ack, err = protocol.ReadRegistration(control, tunnelID)
			control.Close()
			generation = ack.Generation
		}
		stop()
		if err != nil {
			session.Close()
			conn.Close()
			return nil, fmt.Errorf("registration acknowledgement failed: %w", err)
		}
		if c.attachment != nil {
			if err := c.attachment.Acknowledge(generation, tunnelURL); err != nil {
				session.Close()
				conn.Close()
				return nil, err
			}
		}
	}
	if version := resp.Header.Get(protocol.LifecycleHeader); version != "" {
		if version != "1" || generation == "" {
			session.Close()
			conn.Close()
			return nil, errors.New("invalid lifecycle negotiation")
		}
		stop := context.AfterFunc(connectContext, func() { session.Close(); conn.Close() })
		lifecycle, err = session.AcceptStream()
		if err == nil {
			deadline, _ := connectContext.Deadline()
			lifecycle.SetReadDeadline(deadline)
			lifecycleReader = bufio.NewReaderSize(lifecycle, protocol.MaxLifecycleSize)
			_, err = protocol.ReadLifecycle(lifecycleReader, tunnelID, generation, "hello")
			lifecycle.SetReadDeadline(time.Time{})
		}
		stop()
		if err != nil {
			session.Close()
			conn.Close()
			return nil, fmt.Errorf("lifecycle hello failed: %w", err)
		}
	}
	if generation != "" {
		c.emit(LifecycleEvent{Type: "registered", TunnelID: tunnelID, Port: port, URL: tunnelURL, Generation: generation, RegistrationConfirmed: true, LifecycleConfirmed: lifecycle != nil})
	}

	return &TunnelConn{
		ID:          tunnelID,
		Port:        port,
		URL:         tunnelURL,
		wsConn:      conn,
		session:     session,
		latency:     latency,
		status:      "active",
		generation:  generation,
		localHealth: make(map[uint16]HealthObservation),
		lifecycle:   lifecycle, lifecycleReader: lifecycleReader,
	}, nil
}

func (c *Client) handleTunnel(tc *TunnelConn, port uint16) {
	defer tc.cancel()
	defer func() {
		c.mu.Lock()
		if c.tunnels[tc.ID] == tc {
			delete(c.tunnels, tc.ID)
		}
		c.mu.Unlock()
	}()

	reconnectDelay := time.Second
	attempt := 0
	c.watchLifecycle(tc)

	for {
		stream, err := tc.session.AcceptStream()
		if err != nil {
			if tc.ctx.Err() != nil {
				return
			}
			c.mu.Lock()
			wasConnected := tc.status == "active" || tc.status == "draining"
			code := "transport_lost"
			if tc.status == "draining" {
				code = "server_draining"
			}
			generation := tc.generation
			tc.status = "reconnecting"
			c.mu.Unlock()
			if wasConnected {
				c.emit(LifecycleEvent{Type: "disconnected", TunnelID: tc.ID, Port: port, Code: code, Generation: generation})
			}
			attempt++
			delay := reconnectDelay/2 + time.Duration(rand.Int64N(int64(reconnectDelay/2)))
			c.emit(LifecycleEvent{Type: "reconnecting", TunnelID: tc.ID, Port: port, Attempt: attempt, RetryDelayMS: delay.Milliseconds()})

			select {
			case <-tc.ctx.Done():
				return
			case <-time.After(delay):
			}
			reconnectDelay = min(reconnectDelay*2, 30*time.Second)

			// Persistent ownership makes the original ID safe to retry. Switching
			// IDs after failures would bypass a closure received while offline.
			started := time.Now()
			newTC, err := c.createTunnelSession(tc.ctx, tc.ID, port)
			if err != nil {
				c.observer.Operation(telemetry.Reconnect, started, telemetry.Unavailable)
				var rejected *connectionRejected
				if errors.As(err, &rejected) && (rejected.status == http.StatusUnauthorized || rejected.status == http.StatusForbidden || rejected.status == http.StatusConflict || rejected.status == http.StatusGone) {
					c.emit(LifecycleEvent{Type: "failed", TunnelID: tc.ID, Port: port, Code: "connection_rejected"})
					return
				}
				if tc.ctx.Err() != nil {
					return
				}
				continue
			}

			c.mu.Lock()
			if tc.ctx.Err() != nil {
				c.mu.Unlock()
				newTC.session.Close()
				newTC.wsConn.Close()
				return
			}
			tc.wsConn.Close()
			tc.wsConn, tc.session = newTC.wsConn, newTC.session
			tc.latency = newTC.latency
			tc.reconnects++
			tc.URL = newTC.URL
			tc.status = "active"
			tc.generation = newTC.generation
			tc.localHealth = newTC.localHealth
			tc.lifecycle, tc.lifecycleReader = newTC.lifecycle, newTC.lifecycleReader
			c.mu.Unlock()
			c.watchLifecycle(tc)
			c.observer.Track(telemetry.TunnelReconnected)
			c.observer.Operation(telemetry.Reconnect, started, telemetry.Success)
			reconnectDelay = time.Second
			attempt = 0
			continue
		}

		reconnectDelay = time.Second
		attempt = 0
		if tc.dispatcher != nil {
			select {
			case c.streams <- struct{}{}:
				generation := tc.generation
				table := *tc.dispatcher
				table.Dial = func(ctx context.Context, port uint16) (net.Conn, error) {
					start := time.Now()
					conn, err := routing.DialLoopback(ctx, port)
					c.recordTCP(tc.ID, generation, port, start, err)
					return conn, err
				}
				c.workers.Go(func() { defer func() { <-c.streams }(); table.ServeStream(tc.ctx, stream) })
			default:
				stream.SetWriteDeadline(time.Now().Add(time.Second))
				io.WriteString(stream, "HTTP/1.1 503 Service Unavailable\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
				stream.Close()
			}
			continue
		}
		generation := tc.generation
		c.workers.Go(func() {
			s := stream
			defer s.Close()
			started := time.Now()
			outcome := telemetry.Unavailable
			defer func() { c.observer.Operation(telemetry.Forward, started, outcome) }()

			conn, err := routing.DialLoopback(tc.ctx, port)
			c.recordTCP(tc.ID, generation, port, started, err)
			if err != nil {
				return
			}
			defer conn.Close()
			stopCancel := context.AfterFunc(tc.ctx, func() { conn.Close(); s.Close() })
			defer stopCancel()

			done := make(chan struct{}, 2)
			var responseBytes int64
			go func() {
				n, _ := io.Copy(conn, s)
				atomic.AddInt64(&tc.totalSize, n)
				conn.Close()
				done <- struct{}{}
			}()
			go func() {
				n, _ := io.Copy(s, conn)
				responseBytes = n
				atomic.AddInt64(&tc.totalSize, n)
				s.Close()
				done <- struct{}{}
			}()
			<-done
			<-done
			if responseBytes > 0 {
				outcome = telemetry.Success
				tc.forwarded.Do(func() { c.observer.Track(telemetry.TunnelForwarded) })
			}
		})
	}
}

func (c *Client) CloseTunnel(port uint16) error {
	c.mu.RLock()
	id := ""
	for _, tc := range c.tunnels {
		if tc.Port == port {
			if id != "" {
				c.mu.RUnlock()
				return errors.New("ambiguous port; select tunnel ID")
			}
			id = tc.ID
		}
	}
	c.mu.RUnlock()
	if id == "" {
		return nil
	}
	return c.CloseTunnelID(id)
}

func (c *Client) CloseTunnelID(id string) error {
	c.mu.RLock()
	tc := c.tunnels[id]
	c.mu.RUnlock()
	if tc == nil {
		return nil
	}
	if err := c.DeleteTunnel(id); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tunnels[id] == tc {
		tc.cancel()
		tc.session.Close()
		tc.wsConn.Close()
		delete(c.tunnels, id)
	}
	return nil
}

func (c *Client) Close() error {
	return c.CloseContext(context.Background())
}

func (c *Client) CloseContext(ctx context.Context) error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.cancel()
		c.mu.Unlock()
		go func() { c.workers.Wait(); close(c.closeDone) }()
	})
	select {
	case <-c.closeDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) ListTunnels() []*protocol.TunnelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tunnels := make([]*protocol.TunnelInfo, 0, len(c.tunnels))
	for _, tc := range c.tunnels {
		tunnels = append(tunnels, &protocol.TunnelInfo{
			TunnelID:   tc.ID,
			Port:       tc.Port,
			URL:        tc.URL,
			Status:     tc.status,
			Latency:    tc.latency.Round(time.Millisecond).String(),
			Reconnects: tc.reconnects,
			TotalSize:  atomic.LoadInt64(&tc.totalSize),
		})
	}
	sort.Slice(tunnels, func(i, j int) bool {
		if tunnels[i].Port == tunnels[j].Port {
			return tunnels[i].TunnelID < tunnels[j].TunnelID
		}
		return tunnels[i].Port < tunnels[j].Port
	})
	return tunnels
}
