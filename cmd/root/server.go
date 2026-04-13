package root

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services/server"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

type Server struct {
	Configs *configs.ServerConfigs
}

func NewServer() *cobra.Command {
	fc := func(cmd *cobra.Command, args []string) {
		configFile, err1 := cmd.Flags().GetString("config")
		configs_, err2 := configs.NewServer(configFile)
		if err := errors.Join(err1, err2); err != nil {
			cmd.PrintErrf("Failed to load server config, %v\n", err)
			os.Exit(1)
		}
		srv := &Server{Configs: configs_}
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
	if !s.Configs.TLS.Enabled {
		logger.Warn("TLS is disabled; control and work connections will be transmitted in plaintext. Do not expose this server directly to untrusted networks.")
	}

	srv, err := server.NewServer(&server.Params{
		BindPort: s.Configs.BindPort,
		Token:    s.Configs.Token,
		TLS: configs.ServerTLSConfigs{
			Enabled:  s.Configs.TLS.Enabled,
			CAFile:   s.Configs.TLS.CAFile,
			CertFile: s.Configs.TLS.CertFile,
			KeyFile:  s.Configs.TLS.KeyFile,
		},
		IPFilter: configs.ServerIPFilterConfigs{
			Allow: s.Configs.IPFilter.Allow,
			Deny:  s.Configs.IPFilter.Deny,
		},
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	if err := srv.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	logger.Infof("Server started successfully, listening on port: %d, tls=%t", s.Configs.BindPort, s.Configs.TLS.Enabled)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Infof("Received signal %v, server is shutting down", sig)

	srv.Stop()
	logger.Infof("Server has stopped")

	return nil

}
