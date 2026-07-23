package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	sharedconfigs "github.com/xiaotiancaipro/nextunnel/internal/shared/configs"
)

type ConfigSpec struct {
	DefaultPath string
	EnvVar      string
}

func LoadConfig[T any](cmd *cobra.Command, spec ConfigSpec, configsType T) (*T, error) {

	path, err := resolveConfigPath(cmd, spec)
	if err != nil {
		return nil, fmt.Errorf("invalid flags: %w", err)
	}

	file, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid config path %q: %w", path, err)
	}

	c, err := sharedconfigs.Load(configsType, file)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return c, nil

}

func resolveConfigPath(cmd *cobra.Command, spec ConfigSpec) (string, error) {
	if isConfigFlagSet(cmd) {
		return cmd.Flags().GetString("config")
	}
	if spec.EnvVar != "" {
		if env := strings.TrimSpace(os.Getenv(spec.EnvVar)); env != "" {
			return env, nil
		}
	}
	return spec.DefaultPath, nil
}

func isConfigFlagSet(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup("config"); f != nil && f.Changed {
			return true
		}
	}
	return false
}
