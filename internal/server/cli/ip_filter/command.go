package ip_filter

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip-filter",
		Short: "manage IP filtering rules",
	}
	cmd.AddCommand(NewListCommand())
	cmd.AddCommand(NewAddCommand())
	cmd.AddCommand(NewDeleteCommand())
	return cmd
}
