package cli

import (
	"os"
	"os/signal"
	"syscall"
)

type App interface {
	Init() error
	Start() error
	Stop()
}

func RunApp(app App) error {

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		signal.Stop(sigCh)
		return err
	case <-sigCh:
		signal.Stop(sigCh)
		app.Stop()
		return <-errCh
	}

}
