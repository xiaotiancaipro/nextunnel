package services

import (
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"go.uber.org/zap"
)

type IpFilter struct {
	Config *configs.IpFilter
	Logger *zap.Logger
}
