package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

type App interface {
	Init() error
	Start() error
	Stop()
}

func RunApp(cmd *cobra.Command, app App) {

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		signal.Stop(sigCh)
		ExitOnErr(cmd, err)
	case <-sigCh:
		signal.Stop(sigCh)
		app.Stop()
		err := <-errCh
		ExitOnErr(cmd, err)
	}

}
