package cert

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
)

var dir string

func NewDownloadCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "download [name] [cert-id]",
		Short: "download a client TLS certificate to a directory",
		Args:  cobra.ExactArgs(2),
		Run:   downloadRun,
	}
	c.Flags().StringVar(&dir, "dir", "", "output directory (default: stored certificate directory)")
	return c
}

func downloadRun(cmd *cobra.Command, args []string) {
	cfg := utils.LoadServerConfig(cmd)
	utils.ExitOnErr(cmd, utils.DownloadClientCert(cmd, cfg, args[0], args[1], dir))
}
