package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func TestLifecycleNegotiatesSeparateStreamAfterRegistrationEOF(t *testing.T) {
	s, proxy, _ := privacyServer(t)
	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxy.URL, "http")+"/tunnels/connect?tunnel_id=notice&port=3000", http.Header{
		"X-Tunnel-Token": {"synthetic-owner"}, protocol.RegistrationHeader: {"1"}, "X-Burrow-Lifecycle": {"1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if resp.Header.Get("X-Burrow-Lifecycle") != "1" {
		t.Fatal("lifecycle was not negotiated")
	}
	session, err := yamux.Client(&protocol.WsConn{Conn: conn}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	ackStream, err := session.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	ackStream.SetReadDeadline(time.Now().Add(time.Second))
	ack, err := protocol.ReadRegistration(ackStream, "notice")
	ackStream.Close()
	if err != nil {
		t.Fatal(err)
	}
	control, err := session.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	control.SetReadDeadline(time.Now().Add(time.Second))
	var hello map[string]any
	decoder := json.NewDecoder(control)
	if err := decoder.Decode(&hello); err != nil {
		t.Fatal(err)
	}
	if hello["type"] != "hello" || hello["generation"] != ack.Generation {
		t.Fatalf("hello mismatch: %v", hello)
	}
	s.mu.RLock()
	active := s.connections["notice"]
	s.mu.RUnlock()
	if active == nil || active.id != ack.Generation {
		t.Fatal("lifecycle acknowledged an uncommitted generation")
	}
	entered, release := make(chan struct{}), make(chan struct{})
	app := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-release:
			io.WriteString(w, "finished")
		case <-r.Context().Done():
		}
	})}
	defer app.Close()
	go app.Serve(session)
	done := make(chan error, 1)
	go func() {
		req, _ := http.NewRequest("GET", proxy.URL+"/slow", nil)
		req.Host = "notice.example.test"
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		done <- err
	}()
	<-entered
	shutdown := make(chan error, 1)
	go func() { shutdown <- s.Shutdown(t.Context()) }()
	var notice protocol.Lifecycle
	if err := decoder.Decode(&notice); err != nil {
		close(release)
		t.Fatal(err)
	}
	if notice.Type != "draining" || notice.Reason != "server_shutdown" || notice.Generation != ack.Generation || notice.RemainingMS > 15000 {
		close(release)
		t.Fatalf("invalid notice: %+v", notice)
	}
	if session.IsClosed() {
		close(release)
		t.Fatal("notice closed admitted session")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-shutdown; err != nil {
		t.Fatal(err)
	}
}
