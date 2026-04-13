package root

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

const configReloadPollInterval = 2 * time.Second

func watchConfigChanges(ctx context.Context, path string, reload func(source string), logger *logrus.Logger) {

	ticker := time.NewTicker(configReloadPollInterval)
	defer ticker.Stop()

	var lastMod time.Time
	if fi, err := os.Stat(path); err == nil {
		lastMod = fi.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fi, err := os.Stat(path)
			if err != nil {
				logger.Warnf("config watch: stat %s: %v", path, err)
				continue
			}
			mod := fi.ModTime()
			if mod.Equal(lastMod) {
				continue
			}
			lastMod = mod
			reload("config file")
		}
	}

}
