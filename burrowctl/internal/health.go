package internal

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"
	"time"

	"burrow/burrowctl/internal/routing"
)

type HealthObservation struct {
	ObservedAt  time.Time `json:"observed_at"`
	LastSuccess time.Time `json:"last_success,omitempty"`
	Source      string    `json:"source"`
	Port        uint16    `json:"port,omitempty"`
	Route       string    `json:"route,omitempty"`
	Generation  string    `json:"generation,omitempty"`
	Outcome     string    `json:"outcome"`
	Reason      string    `json:"reason"`
	ElapsedMS   int64     `json:"elapsed_ms"`
	Status      int       `json:"status,omitempty"`
	Freshness   string    `json:"freshness"`
}

func (o HealthObservation) fresh(now time.Time) HealthObservation {
	o.Freshness = "fresh"
	if o.ObservedAt.IsZero() {
		o.Freshness = "unknown"
	} else if now.Sub(o.ObservedAt) > 30*time.Second {
		o.Freshness = "stale"
	}
	return o
}

type Diagnostic struct {
	Server       HealthObservation   `json:"server"`
	Targets      []HealthObservation `json:"targets"`
	Registration string              `json:"registration"`
	Application  string              `json:"application"`
	NextSteps    string              `json:"next_steps"`
}

func CheckServer(ctx context.Context, server string) HealthObservation {
	start := time.Now()
	o := HealthObservation{Source: "apex_health", Outcome: "failed", Reason: "server_unreachable"}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	u, err := ServerURL(server)
	if err == nil {
		u.Path = "/health"
		u.RawQuery = ""
		req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		client := NewManagementHTTPClient()
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		resp, requestErr := client.Do(req)
		if requestErr == nil {
			resp.Body.Close()
			o.Status = resp.StatusCode
			o.Reason = "server_http_status"
			if resp.StatusCode == 200 {
				o.Outcome = "succeeded"
				o.Reason = "server_reachable"
			}
		}
	}
	o.ObservedAt = time.Now().UTC()
	o.ElapsedMS = time.Since(start).Milliseconds()
	if o.Outcome == "succeeded" {
		o.LastSuccess = o.ObservedAt
	}
	return o.fresh(time.Now())
}

func CheckTCP(ctx context.Context, port uint16) HealthObservation {
	start := time.Now()
	o := HealthObservation{Source: "explicit_tcp", Port: port, Outcome: "failed", Reason: "tcp_unavailable"}
	if ctx.Err() != nil {
		o.Outcome = "not_checked"
		o.Reason = "budget_exhausted"
		return o.fresh(time.Now())
	}
	conn, err := routing.DialLoopback(ctx, port)
	if err == nil {
		conn.Close()
		o.Outcome = "succeeded"
		o.Reason = "tcp_connected"
	}
	o.ObservedAt = time.Now().UTC()
	o.ElapsedMS = time.Since(start).Milliseconds()
	if err == nil {
		o.LastSuccess = o.ObservedAt
	}
	return o.fresh(time.Now())
}

func Diagnose(parent context.Context, server string, ports []uint16) (Diagnostic, error) {
	report := Diagnostic{Registration: "unavailable_in_standalone_process", Application: "not_probed", Targets: []HealthObservation{}, NextSteps: "Start the explicit ports or check loopback binding; check server connectivity; inspect application logs. TCP acceptance is not HTTP health."}
	if len(ports) == 0 || len(ports) > 32 {
		return report, errors.New("doctor requires 1–32 explicit targets")
	}
	for _, port := range ports {
		if port == 0 {
			return report, errors.New("port must be between 1 and 65535")
		}
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	serverDone := make(chan struct{})
	go func() { report.Server = CheckServer(ctx, server); close(serverDone) }()
	report.Targets = make([]HealthObservation, len(ports))
	var workers sync.WaitGroup
	jobs := make(chan int)
	for range min(4, len(ports)) {
		workers.Go(func() {
			for i := range jobs {
				report.Targets[i] = CheckTCP(ctx, ports[i])
			}
		})
	}
	for i := range ports {
		jobs <- i
	}
	close(jobs)
	workers.Wait()
	<-serverDone
	if parent.Err() != nil {
		return report, parent.Err()
	}
	if report.Server.Outcome != "succeeded" {
		return report, errors.New("server reachability check failed")
	}
	for _, target := range report.Targets {
		if target.Outcome != "succeeded" {
			return report, errors.New("one or more TCP targets failed or were not checked")
		}
	}
	return report, nil
}

// Read-only interactive status: one client-scoped, non-overlapping apex read,
// followed by a five-second pause. It is not an application probe/job executor.
func (c *Client) trackReachability() {
	for {
		o := CheckServer(c.ctx, c.server)
		c.mu.Lock()
		if c.ctx.Err() != nil {
			c.mu.Unlock()
			return
		}
		if o.Outcome != "succeeded" {
			o.LastSuccess = c.reachability.LastSuccess
		}
		c.reachability = o
		for _, tc := range c.tunnels {
			tc.latency = 0
			if o.Outcome == "succeeded" {
				tc.latency = time.Duration(o.ElapsedMS) * time.Millisecond
			}
		}
		c.mu.Unlock()
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-c.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (c *Client) recordTCP(id, generation string, port uint16, start time.Time, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tc := c.tunnels[id]
	if tc == nil || tc.sessionKey() != generation || tc.ctx.Err() != nil {
		return
	}
	previous := tc.localHealth[port]
	o := HealthObservation{ObservedAt: time.Now().UTC(), LastSuccess: previous.LastSuccess, Source: "passive_tcp", Generation: tc.generation, Port: port, ElapsedMS: time.Since(start).Milliseconds(), Outcome: "failed", Reason: "tcp_unavailable"}
	if err == nil {
		o.Outcome = "succeeded"
		o.Reason = "tcp_connected"
		o.LastSuccess = o.ObservedAt
	}
	tc.localHealth[port] = o
}

func (c *Client) Health(id string) (HealthObservation, []HealthObservation) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	local := []HealthObservation{}
	if tc := c.tunnels[id]; tc != nil {
		for _, o := range tc.localHealth {
			local = append(local, o.fresh(time.Now()))
		}
		for _, o := range tc.applicationHealth {
			local = append(local, o.fresh(time.Now()))
		}
	}
	sort.Slice(local, func(i, j int) bool {
		a, b := local[i], local[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		return a.Route < b.Route
	})
	return c.reachability.fresh(time.Now()), local
}

// RecheckTCP accepts only the selected tunnel's existing local configuration.
func (c *Client) RecheckTCP(ctx context.Context, id string, port uint16) HealthObservation {
	c.mu.RLock()
	tc := c.tunnels[id]
	if tc == nil {
		c.mu.RUnlock()
		return HealthObservation{Outcome: "not_checked", Reason: "tunnel_unavailable", Freshness: "unknown"}
	}
	generation := tc.sessionKey()
	allowed := port == tc.Port
	if c.routes != nil {
		allowed = false
		for _, route := range c.routes.Routes {
			allowed = allowed || route.Port == port
		}
	}
	c.mu.RUnlock()
	if !allowed {
		return HealthObservation{Outcome: "not_checked", Reason: "target_not_configured", Freshness: "unknown"}
	}
	o := CheckTCP(ctx, port)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tunnels[id] == tc && tc.sessionKey() == generation && tc.ctx.Err() == nil && ctx.Err() == nil {
		o.Generation = tc.generation
		if o.LastSuccess.IsZero() {
			o.LastSuccess = tc.localHealth[port].LastSuccess
		}
		tc.localHealth[port] = o
	}
	return o
}
