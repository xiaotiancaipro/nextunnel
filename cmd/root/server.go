package root

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	configs_ "github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	server_ "github.com/xiaotiancaipro/nextunnel/internal/server/services"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

type server struct {
	workdir string
	pidFile string
}

func NewServer() *cobra.Command {
	c := &cobra.Command{
		Use:   "server",
		Short: "Manage nextunnel server",
		Args:  cobra.ExactArgs(0),
		Run:   new(server).run,
	}
	c.Flags().StringP("workdir", "w", ".nextunnel", "Working directory")
	c.Flags().StringP("daemon", "d", "", "Daemon control: start, stop, reload")
	return c
}

func (s *server) run(cmd *cobra.Command, _ []string) {

	workdir, err1 := cmd.Flags().GetString("workdir")
	daemonOp, err2 := cmd.Flags().GetString("daemon")
	if err := errors.Join(err1, err2); err != nil {
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	configFile := path.Join(workdir, "nextunnel.toml")
	logFile := path.Join(workdir, "logs", "nextunnel.log")

	configs, err := configs_.NewServer(configFile)
	if err != nil {
		cmd.PrintErrf("Failed to load server config, %v\n", err)
		os.Exit(1)
	}

	logger, err := logger_.New("nextunnel-server", logFile)
	if err != nil {
		cmd.PrintErrf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	s.workdir = workdir
	s.pidFile = path.Join(workdir, "nextunnel.pid")

	daemonOp = strings.ToLower(strings.TrimSpace(daemonOp))
	switch daemonOp {
	case "":
	case "start":
		if err := s.daemonStart(); err != nil {
			cmd.PrintErrf("Daemon start failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Printf("nextunnel server started (pid file %s, log %s)\n", s.pidFile, logFile)
		return
	case "stop":
		if err := s.daemonStop(); err != nil {
			cmd.PrintErrf("Daemon stop failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Println("nextunnel server stop signal sent (SIGTERM)")
		return
	case "reload":
		if err := s.daemonReload(); err != nil {
			cmd.PrintErrf("Daemon reload failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Println("nextunnel server reload signal sent (SIGHUP)")
		return
	default:
		cmd.PrintErrf("--daemon must be start, stop, or reload\n")
		os.Exit(1)
	}

	if !configs.TLS.Enabled {
		logger.Warn("TLS is disabled; control and work connections will be transmitted in plaintext. Do not expose this server directly to untrusted networks.")
	}

	params := &server_.Params{
		BindPort: configs.BindPort,
		Token:    configs.Token,
		TLS:      configs.TLS,
		IPFilter: configs.IPFilter,
		Logger:   logger,
	}
	srv, err := server_.NewServer(params)
	if err != nil {
		cmd.PrintErrf("failed to initialize server: %v", err)
		os.Exit(1)
	}
	if err := srv.Start(); err != nil {
		cmd.PrintErrf("failed to start server: %v", err)
		os.Exit(1)
	}
	logger.Infof("Server started successfully, listening on port: %d, tls=%t", configs.BindPort, configs.TLS.Enabled)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {

		sig := <-sigCh

		if sig == syscall.SIGHUP {
			configsNew, err := configs_.NewServer(configFile)
			if err != nil {
				logger.Warnf("Failed to reload config: %v", err)
				continue
			}
			logger.Info("Applying zero-downtime server config reload")
			if err := srv.ApplyConfig(configsNew); err != nil {
				logger.Errorf("Failed to reload config: %v", err)
				continue
			}
			configs = configsNew
			logger.Info("Server config reloaded successfully")
			continue
		}

		logger.Info("Server is shutting down")
		srv.Stop()
		logger.Info("Server has stopped")

		os.Exit(0)

	}

}

func (s *server) daemonStart() error {

	if err := utils.EnsureStalePidFileCleared(s.pidFile); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	absExe, err := filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	cmd := exec.Command(absExe, "server", "--workdir", s.workdir)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon process: %w", err)
	}

	pid := cmd.Process.Pid
	if err := utils.WritePidFile(s.pidFile, pid); err != nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		return fmt.Errorf("write pid file: %w", err)
	}

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release process: %w", err)
	}

	return nil

}

func (s *server) daemonStop() error {
	pid, err := utils.ReadPidFile(s.pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read pid file: %w", err)
	}
	if !utils.ProcessAlive(pid) {
		_ = os.Remove(s.pidFile)
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal TERM to pid %d: %w", pid, err)
	}
	return nil
}

func (s *server) daemonReload() error {
	pid, err := utils.ReadPidFile(s.pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("pid file not found: %s", s.pidFile)
		}
		return fmt.Errorf("read pid file: %w", err)
	}
	if !utils.ProcessAlive(pid) {
		return fmt.Errorf("no process with pid %d", pid)
	}
	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("signal HUP to pid %d: %w", pid, err)
	}
	return nil
}
