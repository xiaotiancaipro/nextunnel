package server

import (
	"github.com/spf13/cobra"
	args2 "github.com/xiaotiancaipro/nextunnel/cmd/server/args"
	utils2 "github.com/xiaotiancaipro/nextunnel/cmd/server/utils"
)

type client struct{}

func (c *client) new() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "client tools",
	}
	cmd.AddCommand(c.newCreate())
	cmd.AddCommand(c.newCert())
	return cmd
}

func (c *client) newCreate() *cobra.Command {
	var portStart, portEnd int
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client access record",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils2.LoadConfig(cmd)
			utils2.ExitOnErr(cmd, args2.CreateClient(cmd, cfg, posArgs[0], portStart, portEnd))
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
			cfg := utils2.LoadConfig(cmd)
			utils2.ExitOnErr(cmd, args2.ListClientCerts(cmd, cfg, posArgs[0]))
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
			cfg := utils2.LoadConfig(cmd)
			utils2.ExitOnErr(cmd, args2.CreateClientCert(cmd, cfg, posArgs[0], expiresAt))
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
			cfg := utils2.LoadConfig(cmd)
			utils2.ExitOnErr(cmd, args2.DeleteClientCert(cmd, cfg, posArgs[0], posArgs[1]))
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
			cfg := utils2.LoadConfig(cmd)
			utils2.ExitOnErr(cmd, args2.DownloadClientCert(cmd, cfg, posArgs[0], posArgs[1], dir))
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "output directory (default: stored certificate directory)")
	return cmd
}
