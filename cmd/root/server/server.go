package server

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	configs_ "github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	server_ "github.com/xiaotiancaipro/nextunnel/internal/server/services"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

const serverDaemonPidEnvKey = "NEXTUNNEL_SERVER_PIDFILE"

type server struct {
	configs    *configs_.ServerConfigs
	configPath string
	logger     *logrus.Logger
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "server",
		Short: "Manage nextunnel server",
		Args:  cobra.ExactArgs(1),
		Run:   server{}.run,
	}
	c.Flags().String("daemon", "", "Daemon control: start, stop, reload")
	return c
}

func (s *server) run(cmd *cobra.Command, args []string) {

	runDir, err1 := filepath.Abs(args[0])
	daemonOp, err2 := cmd.Flags().GetString("daemon")
	if err := errors.Join(err1, err2); err != nil {
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	configFile := path.Join(runDir, "nextunnel.toml")
	pidFile := path.Join(runDir, "pid", "nextunnel.pid")
	logFile := path.Join(runDir, "logs", "nextunnel.log")

	daemonOp = strings.ToLower(strings.TrimSpace(daemonOp))
	switch daemonOp {
	case "":
	case "start":
		if err := runServerDaemonStart(configFile, pidFile); err != nil {
			cmd.PrintErrf("Daemon start failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Printf("nextunnel server started (pid file %s, log %s)\n", pidFile, logFile)
		return
	case "stop":
		if err := runServerDaemonStop(pidFile); err != nil {
			cmd.PrintErrf("Daemon stop failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Println("nextunnel server stop signal sent (SIGTERM)")
		return
	case "reload":
		if err := runServerDaemonReload(pidFile); err != nil {
			cmd.PrintErrf("Daemon reload failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Println("nextunnel server reload signal sent (SIGHUP)")
		return
	default:
		cmd.PrintErrf("--daemon must be start, stop, or reload\n")
		os.Exit(1)
	}

	configs, err := configs_.NewServer(configFile)
	if err != nil {
		cmd.PrintErrf("Failed to load server config, %v\n", err)
		os.Exit(1)
	}

	s.configs = configs
	s.configPath = configFile
	s.logger = logger_.New("nextunnel-server")
	if err := s.startAndStop(); err != nil {
		cmd.PrintErrf("Server error, %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)

}

func (s *server) startAndStop() error {

	if !s.configs.TLS.Enabled {
		s.logger.Warn("TLS is disabled; control and work connections will be transmitted in plaintext. Do not expose this server directly to untrusted networks.")
	}

	params := &server_.Params{
		BindPort: s.configs.BindPort,
		Token:    s.configs.Token,
		TLS:      s.configs.TLS,
		IPFilter: s.configs.IPFilter,
		Logger:   s.logger,
	}
	srv, err := server_.NewServer(params)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.logger.Infof("Server started successfully, listening on port: %d, tls=%t", s.configs.BindPort, s.configs.TLS.Enabled)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigCh
		if sig == syscall.SIGHUP {
			s.reload(srv, "SIGHUP")
			continue
		}
		s.logger.Infof("Received signal %v, server is shutting down", sig)
		srv.Stop()
		s.logger.Infof("Server has stopped")
		return nil
	}

}

func (s *server) reload(srv *server_.Server, source string) {
	cfg, err := configs_.NewServer(s.configPath)
	if err != nil {
		s.logger.Warnf("%s: failed to reload config: %v", source, err)
		return
	}
	s.logger.Infof("%s: applying zero-downtime server config reload", source)
	if err := srv.ApplyConfig(cfg); err != nil {
		s.logger.Errorf("%s: failed to reload server config: %v", source, err)
		return
	}
	s.configs = cfg
	s.logger.Infof("%s: server config reloaded successfully", source)
}
