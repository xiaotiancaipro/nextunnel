package args

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/certs"
)

func RunGenerateCerts(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if !cmd.Flags().Changed("generate-certs") {
		return false, nil
	}

	out, err := cmd.Flags().GetString("generate-certs")
	if err != nil {
		return false, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return false, nil
	}

	if err := certs.GenerateClientToDir(cfg.Tls.Dir, cfg.Server.Host, out); err != nil {
		return true, err
	}

	abs, err := filepath.Abs(out)
	if err != nil {
		return true, err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s and %s in %s\n", certs.FileClientCert, certs.FileClientKey, abs)
	return true, nil

}
