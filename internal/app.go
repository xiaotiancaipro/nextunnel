package internal

import (
	"fmt"

	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	"go.uber.org/zap"
)

type App struct {
	logger *zap.Logger
	stopCh chan struct{}
}

func NewApp(config *configs.Configs) (*App, error) {

	logger, err := utils.NewLogger(config.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %v", err)
	}

	app := App{
		logger: logger,
	}

	return &app, nil

}

func (a *App) Start() error {

	// TODO

	return nil

}
