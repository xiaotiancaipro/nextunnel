package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

// 消息类型常量
const (
	MsgLogin         byte = 0x01 // client → server: 登录认证
	MsgLoginResp     byte = 0x02 // server → client: 登录响应
	MsgNewProxy      byte = 0x03 // client → server: 注册新代理
	MsgNewProxyResp  byte = 0x04 // server → client: 代理注册响应
	MsgNewWorkConn   byte = 0x05 // server → client: 请求建立工作连接
	MsgStartWorkConn byte = 0x06 // client → server: 工作连接就绪
	MsgPing          byte = 0x07 // client → server: 心跳
	MsgPong          byte = 0x08 // server → client: 心跳响应
)

// LoginMsg client 登录消息
type LoginMsg struct {
	Token string `json:"token"`
}

// LoginRespMsg server 登录响应
type LoginRespMsg struct {
	RunID string `json:"run_id,omitempty"` // 分配给 client 的唯一标识
	Error string `json:"error,omitempty"`
}

// NewProxyMsg client 注册代理消息
type NewProxyMsg struct {
	Name       string `json:"name"`
	Type       string `json:"type"`        // 当前支持 "tcp"
	RemotePort int    `json:"remote_port"` // 服务端监听端口
}

// NewProxyRespMsg server 代理注册响应
type NewProxyRespMsg struct {
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}

// NewWorkConnMsg server 通知 client 建立工作连接
type NewWorkConnMsg struct {
	WorkID    string `json:"work_id"`    // 唯一工作连接标识
	ProxyName string `json:"proxy_name"` // 对应的代理名称
}

// StartWorkConnMsg client 在工作连接上的首条消息，告知 server 此连接属于哪个工作任务
type StartWorkConnMsg struct {
	WorkID string `json:"work_id"`
}

// PingMsg 心跳请求
type PingMsg struct{}

// PongMsg 心跳响应
type PongMsg struct{}

const maxMsgSize = 1 << 20 // 1MB 消息大小上限

// WriteMsg 向连接写入一条消息
// 协议格式: [1 byte: type][4 bytes big-endian: payload length][payload bytes]
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

// ReadMsg 从连接读取一条消息，返回消息类型与原始 payload 字节
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

// Decode 将 payload 解码到目标结构体
func Decode(payload []byte, v interface{}) error {
	return json.Unmarshal(payload, v)
}
