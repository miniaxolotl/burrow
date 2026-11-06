package internal

import (
	"time"

	"burrow/protocol"
)

// One owned reader per negotiated generation. It never touches an application
// stream, never initiates an early parallel reconnect, and cannot update a newer
// session. Stream deadlines enforce the bounded advertised remaining budget.
func (c *Client) watchLifecycle(tc *TunnelConn) {
	c.mu.RLock()
	stream, reader, session, generation := tc.lifecycle, tc.lifecycleReader, tc.session, tc.generation
	c.mu.RUnlock()
	if stream == nil {
		return
	}
	c.workers.Go(func() {
		defer stream.Close()
		noticeID := ""
		for {
			frame, err := protocol.ReadLifecycle(reader, tc.ID, generation, "draining")
			if err != nil {
				session.Close()
				return
			}
			if noticeID != "" {
				if noticeID != frame.NoticeID {
					session.Close()
					return
				}
				continue // Identical duplicate cannot extend the original deadline.
			}
			noticeID = frame.NoticeID
			stream.SetReadDeadline(time.Now().Add(time.Duration(frame.RemainingMS) * time.Millisecond))
			c.mu.Lock()
			if c.tunnels[tc.ID] != tc || tc.session != session || tc.ctx.Err() != nil {
				c.mu.Unlock()
				return
			}
			tc.status = "draining"
			c.mu.Unlock()
			c.emit(LifecycleEvent{Type: "draining", TunnelID: tc.ID, Port: tc.Port, Generation: generation, Code: "server_draining", RemainingMS: frame.RemainingMS, LifecycleConfirmed: true, RegistrationConfirmed: true})
		}
	})
}
