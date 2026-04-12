package apps

import (
	"github.com/sirupsen/logrus"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services"
)

// Server 服务端应用
type Server struct {
	Configs *configs.ServerConfigs
	logger  *logrus.Logger
	srv     *services.Server
}

// Client 客户端应用
type Client struct {
	Configs *configs.ClientConfigs
	logger  *logrus.Logger
	cli     *services.Client
}
