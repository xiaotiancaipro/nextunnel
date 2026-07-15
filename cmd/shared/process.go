package shared

import (
	"os"

	"github.com/spf13/cobra"
)

func ExitOnErr(cmd *cobra.Command, err error) {
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
}
