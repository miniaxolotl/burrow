package internal

import (
	"crypto/rand"
	"sync"
	"sync/atomic"
	"time"

	"burrow/burrowctl/internal/routing"
	"burrow/protocol"
)

type liveHistory struct {
	mu      sync.Mutex
	rows    []*protocol.TunnelLog
	evicted int
	dropped atomic.Uint64
}

type InspectionSnapshot struct {
	Rows       []*protocol.TunnelLog
	Generation string
	Window     string
	Evicted    int
	Dropped    uint64
	Enabled    bool
}

func (c *Client) SetInspect(enabled bool) { c.inspect = enabled }

func (c *Client) Inspection(id string) InspectionSnapshot {
	c.mu.RLock()
	tc := c.tunnels[id]
	if tc == nil {
		c.mu.RUnlock()
		return InspectionSnapshot{Enabled: c.inspect}
	}
	history, generation, window := tc.history, tc.generation, tc.sessionKey()
	c.mu.RUnlock()
	if history == nil {
		return InspectionSnapshot{Generation: generation}
	}
	history.mu.Lock()
	defer history.mu.Unlock()
	return InspectionSnapshot{Rows: append([]*protocol.TunnelLog{}, history.rows...), Generation: generation, Window: window, Evicted: history.evicted, Dropped: history.dropped.Load(), Enabled: true}
}

func (c *Client) recordExchange(id, generation string, e routing.Exchange) {
	c.mu.RLock()
	tc := c.tunnels[id]
	if tc == nil || tc.sessionKey() != generation || tc.ctx.Err() != nil {
		c.mu.RUnlock()
		return
	}
	history := tc.history
	peerGeneration := tc.generation
	c.mu.RUnlock()
	if history == nil {
		return
	}
	if !history.mu.TryLock() {
		history.dropped.Add(1)
		return
	}
	defer history.mu.Unlock()
	ns := e.CompletedAt.Sub(e.StartedAt).Nanoseconds()
	method := e.Method
	if len(method) > 256 {
		method = method[:256]
	}
	entry := &protocol.TunnelLog{ID: rand.Text(), Generation: peerGeneration, Timestamp: e.StartedAt, CompletedAt: e.CompletedAt, Method: method, Path: e.Path, Route: e.Route, DurationNS: &ns, Duration: time.Duration(ns).String(), StatusCode: e.Status, StatusOrigin: e.Origin, Outcome: e.Outcome, Reason: e.Reason}
	var evicted int
	history.rows, evicted = protocol.AppendMetadata(history.rows, entry)
	history.evicted += evicted
}

func (c *Client) recordApplication(id, generation string, route routing.Route, status int, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tc := c.tunnels[id]
	if tc == nil || tc.sessionKey() != generation || tc.ctx.Err() != nil {
		return
	}
	tc.applicationHealth[route.Path] = HealthObservation{Source: "passive_http", Port: route.Port, Route: route.Path, Generation: tc.generation, ObservedAt: at, LastSuccess: at, Outcome: "responded", Reason: "application_response", Status: status}
}

func (tc *TunnelConn) sessionKey() string {
	if tc.epoch != "" {
		return tc.epoch
	}
	return tc.generation
}
