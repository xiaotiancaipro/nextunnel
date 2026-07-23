package configs

import sharedconfigs "github.com/xiaotiancaipro/nextunnel/internal/shared/configs"

const (
	defaultLogFile       = "logs/nextunnel-server.log"
	defaultLogLevel      = "info"
	defaultLogMaxsize    = "100MB"
	defaultLogMaxBackups = 30
	defaultLogMaxAge     = 7
)

func (c *Configs) CheckLogs() error {
	if c.Logs == nil {
		c.Logs = &sharedconfigs.Logs{}
	}
	if c.Logs.File == "" {
		c.Logs.File = defaultLogFile
	}
	if c.Logs.Level == "" {
		c.Logs.Level = defaultLogLevel
	}
	if c.Logs.MaxSize == "" {
		c.Logs.MaxSize = defaultLogMaxsize
	}
	if c.Logs.MaxBackups <= 0 {
		c.Logs.MaxBackups = defaultLogMaxBackups
	}
	if c.Logs.MaxAge <= 0 {
		c.Logs.MaxAge = defaultLogMaxAge
	}
	return nil
}
