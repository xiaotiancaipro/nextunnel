package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	"github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
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

	clientName := strings.TrimSpace(args[0])
	if clientName == "" {
		shared.ExitOnErr(cmd, fmt.Errorf("client name is required"))
	}

	cfg := cli.LoadServerConfig(cmd)
	registry, certService, err := cli.NewClientRegistryAndCertFromConfig(cfg)
	shared.ExitOnErr(cmd, err)

	client, err := registry.GetByName(clientName)
	shared.ExitOnErr(cmd, err)

	certID, err := services.ParseCertID(args[1])
	shared.ExitOnErr(cmd, err)

	certPEM, keyPEM, err := certService.ReadFiles(client.Id, certID)
	shared.ExitOnErr(cmd, err)

	outDir := strings.TrimSpace(dir)
	if outDir == "" {
		outDir, err = cli.CertOutputDir(cfg, clientName, certID.String())
		shared.ExitOnErr(cmd, err)
	} else {
		outDir, err = cli.EnsureOutputDir(outDir)
		shared.ExitOnErr(cmd, err)
	}

	if err := certs.WriteClientPEMToDir(outDir, certPEM, keyPEM); err != nil {
		shared.ExitOnErr(cmd, err)
	}

	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"wrote %s and %s for certificate %q in %s\n",
		certs.FileClientCert,
		certs.FileClientKey,
		certID,
		outDir,
	)

}
