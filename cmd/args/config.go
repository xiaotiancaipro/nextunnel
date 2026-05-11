package args

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-client/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-client/internal/utils"
)

type Config struct{}

func (*Config) New(cmd *cobra.Command) *configs.Configs {

	flag, err := cmd.Flags().GetString("config")
	if err != nil {
		utils.ProcessNotifyDaemonStartFailure(fmt.Errorf("invalid flags: %w", err))
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	file, err := filepath.Abs(flag)
	if err != nil {
		utils.ProcessNotifyDaemonStartFailure(fmt.Errorf("invalid --config: %w", err))
		cmd.PrintErrf("Invalid --config path: %v\n", err)
		os.Exit(1)
	}

	c, err := configs.NewConfigs(file)
	if err != nil {
		utils.ProcessNotifyDaemonStartFailure(fmt.Errorf("load client config: %w", err))
		cmd.PrintErrf("Failed to load client config, %v\n", err)
		os.Exit(1)
	}
	return c

}
