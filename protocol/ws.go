package protocol

import (
	"fmt"

	"github.com/gorilla/websocket"
)

// WsConn wraps *websocket.Conn to implement io.ReadWriter for yamux sessions.
type WsConn struct {
	*websocket.Conn
	buf []byte
}

func (c *WsConn) Read(b []byte) (int, error) {
	if len(c.buf) > 0 {
		n := copy(b, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}
	msgType, msg, err := c.Conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	if msgType != websocket.BinaryMessage {
		return 0, fmt.Errorf("expected binary message")
	}
	n := copy(b, msg)
	if n < len(msg) {
		c.buf = msg[n:]
	}
	return n, nil
}

func (c *WsConn) Write(b []byte) (int, error) {
	if err := c.Conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}
