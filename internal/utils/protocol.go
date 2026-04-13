package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const (
	MsgLogin         byte = 0x01 // client → server
	MsgLoginResp     byte = 0x02 // server → client
	MsgNewProxy      byte = 0x03 // client → server
	MsgNewProxyResp  byte = 0x04 // server → client
	MsgNewWorkConn   byte = 0x05 // server → client
	MsgStartWorkConn byte = 0x06 // client → server
	MsgPing          byte = 0x07 // client → server
	MsgPong          byte = 0x08 // server → client
)

const maxMsgSize = 1 << 20 // 1MB 消息大小上限

type LoginMsg struct {
	Token string `json:"token"`
}

type LoginRespMsg struct {
	RunID string `json:"run_id,omitempty"` // 分配给 client 的唯一标识
	Error string `json:"error,omitempty"`
}

type NewProxyMsg struct {
	Name       string `json:"name"`
	Type       string `json:"type"`        // 当前支持 "tcp"
	RemotePort int    `json:"remote_port"` // 服务端监听端口
}

type NewProxyRespMsg struct {
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}

type NewWorkConnMsg struct {
	WorkID    string `json:"work_id"`    // 唯一工作连接标识
	ProxyName string `json:"proxy_name"` // 对应的代理名称
}

type StartWorkConnMsg struct {
	WorkID string `json:"work_id"`
}

type PingMsg struct{}

type PongMsg struct{}

func WriteMsg(conn net.Conn, msgType byte, payload interface{}) error {

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	buf := make([]byte, 5+len(data))
	buf[0] = msgType
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(data)))
	copy(buf[5:], data)

	_, err = conn.Write(buf)
	return err

}

func ReadMsg(conn net.Conn) (byte, []byte, error) {

	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	msgType := header[0]
	length := binary.BigEndian.Uint32(header[1:5])

	if length > maxMsgSize {
		return 0, nil, fmt.Errorf("消息过大: %d bytes", length)
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
