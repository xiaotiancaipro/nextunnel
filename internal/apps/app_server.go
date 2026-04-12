package apps

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	server2 "github.com/xiaotiancaipro/nextunnel/internal/services"
)

func (s *Server) StartAndStop() error {

	if s.logger == nil {
		s.logger = newLogger("server")
	}

	srv, err := server2.NewServer(&server2.ServerParams{
		BindPort: s.Configs.BindPort,
		Token:    s.Configs.Token,
		Logger:   s.logger,
	})
	if err != nil {
		return fmt.Errorf("初始化服务端失败: %w", err)
	}
	s.srv = srv

	if err := s.srv.Start(); err != nil {
		return fmt.Errorf("启动服务端失败: %w", err)
	}
	s.logger.Infof("服务端启动成功, 监听端口: %d", s.Configs.BindPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	s.logger.Infof("已收到信号 %v, 服务端正在关闭", sig)

	s.srv.Stop()
	s.logger.Infof("服务端已关闭")
	return nil
}

func newLogger(name string) *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	l.WithField("module", name)
	return l
}
