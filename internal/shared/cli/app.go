package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

type App interface {
	Start() error
	Stop()
}

func Run(cmd *cobra.Command, app App) {
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		signal.Stop(sigCh)
		if err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
	case <-sigCh:
		signal.Stop(sigCh)
		app.Stop()
		if err := <-errCh; err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
	}
}
