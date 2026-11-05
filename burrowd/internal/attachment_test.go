package internal

import (
	"burrow/protocol"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPersistentAdmissionIsDurableConditionalAndIdempotent(t *testing.T) {
	s, _, mr := privacyServer(t)
	r := s.registry.redis
	ctx := t.Context()
	if err := r.ReservePersistent(ctx, "saved", "owner"); err != nil {
		t.Fatal(err)
	}
	if err := r.ReserveTunnel(ctx, "saved", "owner"); !errors.Is(err, ErrPersistentRequired) {
		t.Fatalf("legacy reserve bypass: %v", err)
	}
	if err := r.SetTunnel(ctx, "saved", 3000, "owner", "legacy", time.Hour); !errors.Is(err, ErrPersistentRequired) {
		t.Fatalf("legacy register bypass: %v", err)
	}
	gen, err := r.SetPersistent(ctx, "saved", 3000, "owner", "attachment", "", "nonce1", "g1")
	if err != nil || gen != "g1" {
		t.Fatalf("first commit: %q %v", gen, err)
	}
	gen, err = r.SetPersistent(ctx, "saved", 3000, "owner", "attachment", "", "nonce1", "ignored")
	if err != nil || gen != "g1" {
		t.Fatalf("uncertain retry advanced: %q %v", gen, err)
	}
	if _, err := r.SetPersistent(ctx, "saved", 3000, "owner", "attachment", "", "nonce2", "stale"); !errors.Is(err, ErrStaleAttachment) {
		t.Fatalf("stale advanced: %v", err)
	}
	if _, err := r.SetPersistent(ctx, "saved", 3000, "owner", "other", "g1", "nonce2", "takeover"); !errors.Is(err, ErrStaleAttachment) {
		t.Fatalf("different attachment advanced: %v", err)
	}
	mr.FastForward(25 * time.Hour)
	gen, err = r.SetPersistent(ctx, "saved", 3000, "owner", "attachment", "g1", "nonce2", "g2")
	if err != nil || gen != "g2" {
		t.Fatalf("metadata expiry lost fencing: %q %v", gen, err)
	}
	if err := r.RefreshTunnel(ctx, "saved", "g1", time.Hour); err == nil {
		t.Fatal("old generation refreshed")
	}
	r.DeleteTunnel(ctx, "saved", "g1")
	if td, _ := r.GetTunnel(ctx, "saved"); td == nil || td.SessionID != "g2" {
		t.Fatal("old cleanup deleted new generation")
	}
	if err := r.RevokeTunnel(ctx, "saved", "owner"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.SetPersistent(ctx, "saved", 3000, "owner", "attachment", "g2", "nonce3", "g3"); !errors.Is(err, ErrTunnelRevoked) {
		t.Fatalf("revocation bypass: %v", err)
	}
}

func TestPersistentLiveAdmissionRejectsLegacyAndCompetingAttachments(t *testing.T) {
	s, proxy, _ := privacyServer(t)
	dial := func(expected, nonce string, persistent bool) (*yamux.Session, string, int) {
		h := http.Header{"X-Tunnel-Token": {"synthetic-owner"}, protocol.RegistrationHeader: {"1"}}
		if persistent {
			h.Set("X-Burrow-Attachment", "1")
			h.Set("X-Burrow-Attachment-ID", "attachment1")
			h.Set("X-Burrow-Attempt", nonce)
			h.Set("X-Burrow-Expected-Generation", expected)
		}
		conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxy.URL, "http")+"/tunnels/connect?tunnel_id=saved&port=3000", h)
		if err != nil {
			if resp == nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			return nil, "", resp.StatusCode
		}
		session, err := yamux.Client(&protocol.WsConn{Conn: conn}, nil)
		if err != nil {
			t.Fatal(err)
		}
		control, err := session.AcceptStream()
		if err != nil {
			session.Close()
			return nil, "", 101
		}
		control.SetReadDeadline(time.Now().Add(time.Second))
		ack, err := protocol.ReadRegistration(control, "saved")
		control.Close()
		if err != nil {
			session.Close()
			return nil, "", 101
		}
		return session, ack.Generation, 101
	}
	first, generation, _ := dial("", "attempt1", true)
	if first == nil {
		t.Fatal("no first attachment")
	}
	defer first.Close()
	if other, _, status := dial("", "attempt1", true); status != http.StatusLocked {
		if other != nil {
			other.Close()
		}
		t.Fatalf("live nonce retry status %d", status)
	}
	if other, _, status := dial("", "", false); status != http.StatusConflict {
		if other != nil {
			other.Close()
		}
		t.Fatalf("legacy bypass status %d", status)
	}
	first.Close()
	deadline := time.Now().Add(time.Second)
	for {
		s.mu.RLock()
		active := s.connections["saved"]
		s.mu.RUnlock()
		if active == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("disconnect did not clear live registry")
		}
		time.Sleep(time.Millisecond)
	}
	resumed, same, _ := dial("", "attempt1", true)
	if resumed == nil || same != generation {
		t.Fatal("uncertain admission not reconciled")
	}
	resumed.Close()
}
