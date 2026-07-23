package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/client/cert"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	sharedstring "github.com/xiaotiancaipro/nextunnel/internal/shared/string"
)

var dir string

func NewDownloadCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "download [name] [cert-id]",
		Short: "download a client TLS certificate to a directory",
		Args:  cobra.ExactArgs(2),
		RunE:  downloadRun,
	}
	c.Flags().StringVar(&dir, "dir", "", "output directory (default: stored certificate directory)")
	return c
}

func downloadRun(cmd *cobra.Command, args []string) error {

	clientName := strings.TrimSpace(args[0])
	if clientName == "" {
		return fmt.Errorf("client name is required")
	}

	cfg, err := cli.LoadServerConfig(cmd)
	if err != nil {
		return err
	}

	registry, certService, err := cli.NewClientRegistryAndCertFromConfig(cfg)
	if err != nil {
		return err
	}
	defer cli.CloseDatabase(registry.Database)

	client, err := registry.GetByName(clientName)
	if err != nil {
		return err
	}

	certID, err := sharedstring.ParseUUID(args[1])
	if err != nil {
		return err
	}

	certPEM, keyPEM, err := certService.ReadFiles(client.Id, certID)
	if err != nil {
		return err
	}

	outDir := strings.TrimSpace(dir)
	if outDir == "" {
		outDir, err = cert.OutputDir(cfg, clientName, certID.String())
	} else {
		outDir, err = cert.EnsureOutputDir(outDir)
	}
	if err != nil {
		return err
	}

	if err := sharedcerts.WriteClientPEMToDir(outDir, certPEM, keyPEM); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"wrote %s and %s for certificate %q in %s\n",
		sharedcerts.FileClientCert,
		sharedcerts.FileClientKey,
		certID,
		outDir,
	)

	return nil

}
