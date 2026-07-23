package cli

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

const (
	ClientDefaultConfigPath = "nextunnel-client.toml"
	ClientEnvConfigPath     = "NEXTUNNEL_CLIENT_CONFIG"
)

func LoadClientConfig(cmd *cobra.Command) (*configs.Configs, error) {
	spec := sharedcli.ConfigSpec{
		DefaultPath: ClientDefaultConfigPath,
		EnvVar:      ClientEnvConfigPath,
	}
	c, err := sharedcli.LoadConfig(cmd, spec, configs.Configs{})
	if err != nil {
		return nil, err
	}
	checks := []func() error{
		c.CheckCert,
		c.CheckClient,
		c.CheckLogs,
		c.CheckServer,
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return nil, err
		}
	}
	return c, nil
}
