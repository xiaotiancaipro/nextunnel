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
			cmd.PrintErrf("加载服务端配置失败, %v\n", err)
			os.Exit(1)
		}
		server := &Server{Configs: configs_}
		if err := server.StartAndStop(); err != nil {
			cmd.PrintErrf("服务端异常, %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	c := &cobra.Command{
		Use:   "server",
		Short: "启动内网穿透服务端 (server)",
		Args:  cobra.ExactArgs(0),
		Run:   fc,
	}
	c.Flags().StringP("config", "c", "config/nextunnel-server.toml", "服务端配置文件路径")
	return c
}

func (s *Server) StartAndStop() error {

	logger := utils.NewLogger("server")

	server, err := services.NewServer(&services.ServerParams{
		BindPort: s.Configs.BindPort,
		Token:    s.Configs.Token,
		Logger:   logger,
	})
	if err != nil {
		return fmt.Errorf("初始化服务端失败: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("启动服务端失败: %w", err)
	}
	logger.Infof("服务端启动成功, 监听端口: %d", s.Configs.BindPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Infof("已收到信号 %v, 服务端正在关闭", sig)

	server.Stop()
	logger.Infof("服务端已关闭")

	return nil

}
