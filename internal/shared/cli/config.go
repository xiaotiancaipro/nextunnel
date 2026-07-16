package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type ConfigSpec struct {
	DefaultPath string
	EnvVar      string
}

func LoadConfig[T any](cmd *cobra.Command, spec ConfigSpec, loader func(path string) (*T, error), failMsg string) *T {

	path, err := resolveConfigPath(cmd, spec)
	if err != nil {
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	file, err := filepath.Abs(path)
	if err != nil {
		cmd.PrintErrf("Invalid config path %q: %v\n", path, err)
		os.Exit(1)
	}

	c, err := loader(file)
	if err != nil {
		cmd.PrintErrf("%s, %v\n", failMsg, err)
		os.Exit(1)
	}

	return c

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
