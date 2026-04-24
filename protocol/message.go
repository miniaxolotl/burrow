package protocol

import (
	"encoding/binary"
	"fmt"
)

type MessageType uint8

const (
	MsgRegister       MessageType = 0x01
	MsgHeartbeat      MessageType = 0x02
	MsgData           MessageType = 0x03
	MsgClose          MessageType = 0x04
	MsgRegisterResp   MessageType = 0x05
	MsgError          MessageType = 0x06
	MsgTunnelList     MessageType = 0x07
	MsgTunnelRevoke   MessageType = 0x08
	MsgInspectLog     MessageType = 0x09
	MsgStatus        MessageType = 0x0A
)

type Message struct {
	Type     MessageType
	StreamID uint32
	Length   uint32
	Payload  []byte
}

func (m *Message) Serialize() []byte {
	buf := make([]byte, 9+m.Length)
	buf[0] = byte(m.Type)
	binary.BigEndian.PutUint32(buf[1:5], m.StreamID)
	binary.BigEndian.PutUint32(buf[5:9], m.Length)
	copy(buf[9:], m.Payload)
	return buf
}

func DeserializeMessage(data []byte) (*Message, error) {
	if len(data) < 9 {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}
	m := &Message{
		Type:     MessageType(data[0]),
		StreamID: binary.BigEndian.Uint32(data[1:5]),
		Length:   binary.BigEndian.Uint32(data[5:9]),
	}
	if m.Length > 0 {
		payloadStart := 9
		payloadEnd := payloadStart + int(m.Length)
		if len(data) < payloadEnd {
			return nil, fmt.Errorf("payload length mismatch: expected %d, got %d", m.Length, len(data)-9)
		}
		m.Payload = data[payloadStart:payloadEnd]
	}
	return m, nil
}

type RegisterRequest struct {
	TunnelID string
	Port    uint16
}

func (r *RegisterRequest) Serialize() []byte {
	tunnelIDLen := uint16(len(r.TunnelID))
	buf := make([]byte, 2+tunnelIDLen+2)
	binary.BigEndian.PutUint16(buf[0:2], tunnelIDLen)
	copy(buf[2:2+tunnelIDLen], r.TunnelID)
	binary.BigEndian.PutUint16(buf[2+tunnelIDLen:4+tunnelIDLen], r.Port)
	return buf
}

func DeserializeRegisterRequest(data []byte) (*RegisterRequest, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("register request too short")
	}
	tunnelIDLen := binary.BigEndian.Uint16(data[0:2])
	if len(data) < 4+int(tunnelIDLen) {
		return nil, fmt.Errorf("invalid register request data")
	}
	tunnelID := string(data[2 : 2+tunnelIDLen])
	port := binary.BigEndian.Uint16(data[2+tunnelIDLen : 4+tunnelIDLen])
	return &RegisterRequest{
		TunnelID: tunnelID,
		Port:    port,
	}, nil
}

type RegisterResponse struct {
	TunnelID string
	URL     string
}

func (r *RegisterResponse) Serialize() []byte {
	tunnelIDLen := uint16(len(r.TunnelID))
	urlLen := uint16(len(r.URL))
	buf := make([]byte, 2+tunnelIDLen+2+urlLen)
	binary.BigEndian.PutUint16(buf[0:2], tunnelIDLen)
	copy(buf[2:2+tunnelIDLen], r.TunnelID)
	binary.BigEndian.PutUint16(buf[2+tunnelIDLen:4+tunnelIDLen], urlLen)
	copy(buf[4+tunnelIDLen:], r.URL)
	return buf
}

func DeserializeRegisterResponse(data []byte) (*RegisterResponse, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("register response too short")
	}
	tunnelIDLen := binary.BigEndian.Uint16(data[0:2])
	if len(data) < 4+int(tunnelIDLen) {
		return nil, fmt.Errorf("invalid register response data")
	}
	tunnelID := string(data[2 : 2+tunnelIDLen])
	offset := 2 + tunnelIDLen
	urlLen := binary.BigEndian.Uint16(data[offset : offset+2])
	if len(data) < int(offset)+2+int(urlLen) {
		return nil, fmt.Errorf("invalid register response: url length mismatch")
	}
	url := string(data[offset+2 : offset+2+urlLen])
	return &RegisterResponse{
		TunnelID: tunnelID,
		URL:     url,
	}, nil
}

type ErrorMessage struct {
	Code    uint16
	Message string
}

func (e *ErrorMessage) Serialize() []byte {
	msgLen := uint16(len(e.Message))
	buf := make([]byte, 2+msgLen)
	binary.BigEndian.PutUint16(buf[0:2], e.Code)
	copy(buf[2:], e.Message)
	return buf
}

func DeserializeErrorMessage(data []byte) (*ErrorMessage, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("error message too short")
	}
	code := binary.BigEndian.Uint16(data[0:2])
	message := string(data[2:])
	return &ErrorMessage{
		Code:    code,
		Message: message,
	}, nil
}

type TunnelInfo struct {
	TunnelID string
	Port     uint16
	URL      string
	Status   string
}

type TunnelListResponse struct {
	Tunnels []TunnelInfo
}

func (r *TunnelListResponse) Serialize() []byte {
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, uint32(len(r.Tunnels)))
	buf := count

	for _, t := range r.Tunnels {
		tid := []byte(t.TunnelID)
		url := []byte(t.URL)
		status := []byte(t.Status)
		// layout: 2(tidLen) + tid + 2(port) + 2(urlLen) + url + 2(statusLen) + status
		entry := make([]byte, 2+len(tid)+2+2+len(url)+2+len(status))
		off := 0
		binary.BigEndian.PutUint16(entry[off:], uint16(len(tid)))
		off += 2
		copy(entry[off:], tid)
		off += len(tid)
		binary.BigEndian.PutUint16(entry[off:], t.Port)
		off += 2
		binary.BigEndian.PutUint16(entry[off:], uint16(len(url)))
		off += 2
		copy(entry[off:], url)
		off += len(url)
		binary.BigEndian.PutUint16(entry[off:], uint16(len(status)))
		off += 2
		copy(entry[off:], status)
		buf = append(buf, entry...)
	}
	return buf
}

type StatusMessage struct {
	TunnelID  string
	Port      uint16
	Latency   uint32
	Reconnects uint32
}

func (s *StatusMessage) Serialize() []byte {
	tunnelIDLen := uint16(len(s.TunnelID))
	buf := make([]byte, 2+tunnelIDLen+2+4+4)
	binary.BigEndian.PutUint16(buf[0:2], tunnelIDLen)
	copy(buf[2:2+tunnelIDLen], s.TunnelID)
	binary.BigEndian.PutUint16(buf[2+tunnelIDLen:4+tunnelIDLen], s.Port)
	binary.BigEndian.PutUint32(buf[4+tunnelIDLen:8+tunnelIDLen], s.Latency)
	binary.BigEndian.PutUint32(buf[8+tunnelIDLen:12+tunnelIDLen], s.Reconnects)
	return buf
}

type InspectLogMessage struct {
	TunnelID string
	Entries  []LogEntry
}

type LogEntry struct {
	Timestamp int64
	Type      string
	Message   string
}
