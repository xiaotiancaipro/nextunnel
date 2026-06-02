package args

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/certs"
)

func GenerateCerts(cmd *cobra.Command, cfg *configs.Configs, out string) error {

	out = strings.TrimSpace(out)
	if out == "" {
		return fmt.Errorf("output directory is required")
	}

	if err := certs.GenerateClientToDir(cfg.Tls.Dir, cfg.Server.Host, out); err != nil {
		return err
	}

	abs, err := filepath.Abs(out)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s and %s in %s\n", certs.FileClientCert, certs.FileClientKey, abs)
	return nil

}
