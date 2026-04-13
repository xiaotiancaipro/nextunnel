package root

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
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
		server := &Server{Configs: configs_}
		if err := server.Run(); err != nil {
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

	logger := utils.NewLogger("server")

	server, err := services.NewServer(&services.ServerParams{
		BindPort: s.Configs.BindPort,
		Token:    s.Configs.Token,
		Logger:   logger,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	logger.Infof("Server started successfully, listening on port: %d", s.Configs.BindPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Infof("Received signal %v, server is shutting down", sig)

	server.Stop()
	logger.Infof("Server has stopped")

	return nil

}
