package root

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services/server"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

type Server struct {
	Configs    *configs.ServerConfigs
	ConfigPath string
}

func NewServer() *cobra.Command {
	fc := func(cmd *cobra.Command, args []string) {
		configFile, err1 := cmd.Flags().GetString("config")
		configs_, err2 := configs.NewServer(configFile)
		if err := errors.Join(err1, err2); err != nil {
			cmd.PrintErrf("Failed to load server config, %v\n", err)
			os.Exit(1)
		}
		srv := &Server{Configs: configs_, ConfigPath: configFile}
		if err := srv.Run(); err != nil {
			cmd.PrintErrf("Server error, %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	c := &cobra.Command{
		Use:   "server",
		Short: "Start nextunnel server",
		Args:  cobra.ExactArgs(0),
		Run:   fc,
	}
	c.Flags().StringP("config", "c", "server.toml", "Path to server config file")
	return c
}

func (s *Server) Run() error {

	logger := logger_.NewLogger("server")
	srv, err := s.startServer(s.Configs, logger)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reloadServer := func(source string) {
		cfg, err := configs.NewServer(s.ConfigPath)
		if err != nil {
			logger.Warnf("%s: failed to reload config: %v", source, err)
			return
		}
		logger.Infof("%s: applying zero-downtime server config reload", source)
		if err := srv.ApplyConfig(cfg); err != nil {
			logger.Errorf("%s: failed to reload server config: %v", source, err)
			return
		}
		s.Configs = cfg
		logger.Infof("%s: server config reloaded successfully", source)
	}

	go watchConfigChanges(ctx, s.ConfigPath, reloadServer, logger)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigCh
		if sig == syscall.SIGHUP {
			reloadServer("SIGHUP")
			continue
		}
		logger.Infof("Received signal %v, server is shutting down", sig)
		cancel()
		srv.Stop()
		logger.Infof("Server has stopped")
		return nil
	}

}

func (s *Server) startServer(cfg *configs.ServerConfigs, logger *logrus.Logger) (*server.Server, error) {

	if !cfg.TLS.Enabled {
		logger.Warn("TLS is disabled; control and work connections will be transmitted in plaintext. Do not expose this server directly to untrusted networks.")
	}

	srv, err := server.NewServer(&server.Params{
		BindPort: cfg.BindPort,
		Token:    cfg.Token,
		TLS: configs.ServerTLSConfigs{
			Enabled:  cfg.TLS.Enabled,
			CAFile:   cfg.TLS.CAFile,
			CertFile: cfg.TLS.CertFile,
			KeyFile:  cfg.TLS.KeyFile,
		},
		IPFilter: configs.ServerIPFilterConfigs{
			Allow: cfg.IPFilter.Allow,
			Deny:  cfg.IPFilter.Deny,
		},
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}
	if err := srv.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	logger.Infof("Server started successfully, listening on port: %d, tls=%t", cfg.BindPort, cfg.TLS.Enabled)
	return srv, nil

}
