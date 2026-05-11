package services

import (
	"fmt"
	"net"
	"time"

	"github.com/xiaotiancaipro/nextunnel-client/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-client/internal/utils"
	"go.uber.org/zap"
)

type Client struct {
	config *configs.Client
	logger *zap.Logger
}

func NewClient(config *configs.Configs, logger *zap.Logger) *Client {
	return &Client{
		config: config.Client,
		logger: logger,
	}
}

func (c *Client) Login(conn net.Conn) error {
	payload := utils.LoginMsg{
		Id:    c.config.Id,
		Token: c.config.Token,
	}
	if err := utils.WriteMsg(conn, utils.MsgLogin, payload); err != nil {
		c.logger.Error(fmt.Sprintf("failed to write login msg: %v", err))
		return fmt.Errorf("failed to send LoginMsg")
	}
	return nil
}

func (c *Client) LoginResponse(conn net.Conn) (*string, error) {

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		c.logger.Error(fmt.Sprintf("failed to read login msg: %v", err))
		return nil, fmt.Errorf("failed to read LoginResp")
	}
	if msgType != utils.MsgLoginResp {
		c.logger.Error(fmt.Sprintf("invalid login msg type: %v", msgType))
		return nil, fmt.Errorf("expected LoginResp")
	}

	var loginResp utils.LoginRespMsg
	if err := utils.Decode(payload, &loginResp); err != nil {
		c.logger.Error(fmt.Sprintf("failed to decode LoginResp: %v", err))
		return nil, fmt.Errorf("failed to parse LoginResp")
	}
	if loginResp.Error != "" {
		c.logger.Error(fmt.Sprintf("login response error: %v", loginResp.Error))
		return nil, fmt.Errorf("login error")
	}

	return &loginResp.RunID, nil

}
