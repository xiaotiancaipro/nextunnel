package services

import (
	"fmt"
	"net"

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
		ClientID: c.config.Id,
		Token:    c.config.Token,
	}
	if err := utils.WriteMsg(conn, utils.MsgLogin, payload); err != nil {
		c.logger.Error(fmt.Sprintf("failed to write login msg: %v", err))
		return fmt.Errorf("failed to send LoginMsg")
	}
	return nil
}
