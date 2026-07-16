package cert

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "cert",
		Short: "manage client TLS certificates",
	}
	c.AddCommand(NewListCommand())
	c.AddCommand(NewCreateCommand())
	c.AddCommand(NewDeleteCommand())
	c.AddCommand(NewDownloadCommand())
	return c
}
