package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const (
	MsgLogin           byte = 0x01
	MsgLoginResp       byte = 0x02
	MsgApplyConfig     byte = 0x03
	MsgApplyConfigResp byte = 0x04
	MsgNewWorkConn     byte = 0x05
	MsgStartWorkConn   byte = 0x06
	MsgPing            byte = 0x07
	MsgPong            byte = 0x08
)

const maxMsgSize = 1 << 20 // 1MB max message size

type LoginMsg struct {
	ClientID string `json:"client_id"`
	Token    string `json:"token"`
}

type LoginRespMsg struct {
	RunID string `json:"run_id,omitempty"` // unique identifier assigned to the client
	Error string `json:"error,omitempty"`
}

type ApplyConfigMsg struct {
	Proxies []ApplyConfigProxyMsg `json:"proxies"`
}

type ApplyConfigProxyMsg struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	RemotePort int    `json:"remote_port"`
}

type ApplyConfigRespMsg struct {
	Error string `json:"error,omitempty"`
}

type NewWorkConnMsg struct {
	WorkID    string `json:"work_id"`    // unique work connection identifier
	ProxyName string `json:"proxy_name"` // corresponding proxy name
}

type StartWorkConnMsg struct {
	WorkID string `json:"work_id"`
}

type PingMsg struct{}

type PongMsg struct{}

func WriteMsg(conn net.Conn, msgType byte, payload interface{}) error {

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	buf := make([]byte, 5+len(data))
	buf[0] = msgType
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(data)))
	copy(buf[5:], data)

	if _, err := io.Copy(conn, bytes.NewReader(buf)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil

}

func ReadMsg(conn net.Conn) (byte, []byte, error) {

	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	msgType := header[0]
	length := binary.BigEndian.Uint32(header[1:5])

	if length > maxMsgSize {
		return 0, nil, fmt.Errorf("message too large: %d bytes", length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, nil, err
	}

	return msgType, payload, nil

}

func Decode(payload []byte, v interface{}) error {
	return json.Unmarshal(payload, v)
}
