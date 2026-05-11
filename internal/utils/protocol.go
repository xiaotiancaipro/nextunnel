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
	MsgLogin byte = 0x01
)

type LoginMsg struct {
	ClientID string `json:"client_id"`
	Token    string `json:"token"`
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
