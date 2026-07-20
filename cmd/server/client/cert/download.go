package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
	sharedstring "github.com/xiaotiancaipro/nextunnel/internal/shared/string"
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
		sharedcli.ExitOnErr(cmd, fmt.Errorf("client name is required"))
	}

	cfg := cli.LoadServerConfig(cmd)
	registry, certService, err := cli.NewClientRegistryAndCertFromConfig(cfg)
	sharedcli.ExitOnErr(cmd, err)
	defer cli.CloseDatabase(registry.Database)

	client, err := registry.GetByName(clientName)
	cli.ExitOnDBErr(cmd, err, registry.Database)

	certID, err := sharedstring.ParseUUID(args[1])
	cli.ExitOnDBErr(cmd, err, registry.Database)

	certPEM, keyPEM, err := certService.ReadFiles(client.Id, certID)
	cli.ExitOnDBErr(cmd, err, registry.Database)

	outDir := strings.TrimSpace(dir)
	if outDir == "" {
		outDir, err = cli.CertOutputDir(cfg, clientName, certID.String())
		cli.ExitOnDBErr(cmd, err, registry.Database)
	} else {
		outDir, err = cli.EnsureOutputDir(outDir)
		cli.ExitOnDBErr(cmd, err, registry.Database)
	}

	if err := sharedcerts.WriteClientPEMToDir(outDir, certPEM, keyPEM); err != nil {
		cli.ExitOnDBErr(cmd, err, registry.Database)
	}

	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"wrote %s and %s for certificate %q in %s\n",
		sharedcerts.FileClientCert,
		sharedcerts.FileClientKey,
		certID,
		outDir,
	)

}
