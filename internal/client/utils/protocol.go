package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const MsgMaxSize = 1 << 20

const (
	MsgLogin         byte = 0x01
	MsgProxiesApply  byte = 0x02
	MsgNewWorkConn   byte = 0x03
	MsgHeartbeat     byte = 0x04
	MsgStartWorkConn byte = 0x05
)

const (
	MsgLoginResp        byte = 0x11
	MsgProxiesApplyResp byte = 0x12
	MsgHeartbeatResp    byte = 0x13
)

type LoginMsg struct {
	Id string `json:"id"`
}

type LoginRespMsg struct {
	RunID string `json:"run_id,omitempty"`
	Error string `json:"error,omitempty"`
}

type ProxiesApplyMsg struct {
	Proxies []ProxiesApplyMsgItem `json:"proxies"`
}

type ProxiesApplyMsgItem struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	RemotePort int    `json:"remote_port"`
	LocalIP    string `json:"local_ip"`
	LocalPort  int    `json:"local_port"`
}

type ProxiesApplyRespMsg struct {
	Error string `json:"error,omitempty"`
}

type NewWorkConnMsg struct {
	WorkID    string `json:"work_id"`
	ProxyName string `json:"proxy_name"`
}

type HeartbeatMsg struct{}

type HeartbeatRespMsg struct{}

type StartWorkConnMsg struct {
	WorkID string `json:"work_id"`
}

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

	if length > MsgMaxSize {
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
