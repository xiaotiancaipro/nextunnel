package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/utils"
)

type client struct{}

func (c *client) new() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "client tools",
	}
	cmd.AddCommand(c.newCreate())
	cmd.AddCommand(c.newCert())
	cmd.AddCommand(c.newGenerateCerts())
	return cmd
}

func (c *client) newCreate() *cobra.Command {
	var portStart, portEnd int
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client access record",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.CreateClient(cmd, cfg, posArgs[0], portStart, portEnd))
		},
	}
	cmd.Flags().IntVar(&portStart, "port-start", 0, "inclusive start of allocated remote port range")
	cmd.Flags().IntVar(&portEnd, "port-end", 0, "inclusive end of allocated remote port range")
	return cmd
}

func (c *client) newCert() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "manage client TLS certificates",
	}
	cmd.AddCommand(c.newCertList())
	cmd.AddCommand(c.newCertCreate())
	cmd.AddCommand(c.newCertDelete())
	cmd.AddCommand(c.newCertDownload())
	return cmd
}

func (c *client) newCertList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [name]",
		Short: "list certificates for a client",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.ListClientCerts(cmd, cfg, posArgs[0]))
		},
	}
	return cmd
}

func (c *client) newCertCreate() *cobra.Command {
	var expiresAt string
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client TLS certificate",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.CreateClientCert(cmd, cfg, posArgs[0], expiresAt))
		},
	}
	cmd.Flags().StringVar(&expiresAt, "expires-at", "", "certificate expiry time in RFC3339 format (default: never expires)")
	return cmd
}

func (c *client) newCertDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [name] [cert-id]",
		Short: "delete a client TLS certificate",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.DeleteClientCert(cmd, cfg, posArgs[0], posArgs[1]))
		},
	}
	return cmd
}

func (c *client) newCertDownload() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "download [name] [cert-id]",
		Short: "download a client TLS certificate to a directory",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.DownloadClientCert(cmd, cfg, posArgs[0], posArgs[1], dir))
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "output directory (default: stored certificate directory)")
	return cmd
}

func (c *client) newGenerateCerts() *cobra.Command {
	var dir, expiresAt string
	cmd := &cobra.Command{
		Use:        "generate-certs [name]",
		Short:      "generate client TLS certificates (alias for cert create)",
		Deprecated: "use \"client cert create\" instead",
		Args:       cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.GenerateCerts(cmd, cfg, dir, posArgs[0], expiresAt))
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "output directory for client certificates (default: [cert].dir/clients/<name>/<cert-id>)")
	cmd.Flags().StringVar(&expiresAt, "expires-at", "", "certificate expiry time in RFC3339 format (default: never expires)")
	return cmd
}
