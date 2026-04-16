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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	configs_ "github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	server_ "github.com/xiaotiancaipro/nextunnel/internal/server/services"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

const serverDaemonPidEnvKey = "NEXTUNNEL_SERVER_PIDFILE"

type server struct {
	configs    *configs_.ServerConfigs
	configPath string
	logger     *logrus.Logger
}

func NewServer() *cobra.Command {
	c := &cobra.Command{
		Use:   "server",
		Short: "Manage nextunnel server",
		Args:  cobra.ExactArgs(0),
		Run:   server{}.run,
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
	pidFile := path.Join(workdir, "nextunnel.pid")
	logFile := path.Join(workdir, "logs", "nextunnel.log")

	daemonOp = strings.ToLower(strings.TrimSpace(daemonOp))
	switch daemonOp {
	case "":
	case "start":
		if err := daemonStart(configFile, pidFile); err != nil {
			cmd.PrintErrf("Daemon start failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Printf("nextunnel server started (pid file %s, log %s)\n", pidFile, logFile)
		return
	case "stop":
		if err := daemonStop(pidFile); err != nil {
			cmd.PrintErrf("Daemon stop failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Println("nextunnel server stop signal sent (SIGTERM)")
		return
	case "reload":
		if err := daemonReload(pidFile); err != nil {
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
			cfg, err := configs_.NewServer(s.configPath)
			if err != nil {
				s.logger.Warnf("Failed to reload config: %v", err)
				continue
			}
			s.logger.Info("Applying zero-downtime server config reload")
			if err := srv.ApplyConfig(cfg); err != nil {
				s.logger.Errorf("Failed to reload config: %v", err)
				continue
			}
			s.configs = cfg
			s.logger.Info("Server config reloaded successfully")
			continue
		}

		s.logger.Info("Server is shutting down")
		srv.Stop()
		s.logger.Info("Server has stopped")

		return nil

	}

}

func daemonStart(configFile string, pidFile string) error {

	if err := utils.EnsureStalePidFileCleared(pidFile); err != nil {
		return err
	}

	absConfig, err := filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	absExe, err := filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	logPath := utils.LogPathBesideConfig(absConfig)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}

	cmd := exec.Command(absExe, "server", "--config", absConfig)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", serverDaemonPidEnvKey, pidFile))
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start daemon process: %w", err)
	}
	_ = logFile.Close()
	pid := cmd.Process.Pid
	if err := utils.WritePidFile(pidFile, pid); err != nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		return fmt.Errorf("write pid file: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release process: %w", err)
	}
	return nil
}

func daemonStop(pidPath string) error {
	pid, err := utils.ReadPidFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read pid file: %w", err)
	}
	if !utils.ProcessAlive(pid) {
		_ = os.Remove(pidPath)
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal TERM to pid %d: %w", pid, err)
	}
	return nil
}

func daemonReload(pidPath string) error {
	pid, err := utils.ReadPidFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("pid file not found: %s", pidPath)
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
