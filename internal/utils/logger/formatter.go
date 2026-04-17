package logger

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

var callerPathRoots = [...]string{
	"cmd",
	"internal",
}

type Formatter struct {
	logrus.TextFormatter
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	file := ""
	line := 0
	if entry.Caller != nil {
		file = trimCallerPath(entry.Caller.File)
		line = entry.Caller.Line
	}
	format := fmt.Sprintf(
		"%s - %s - %s - %d - %s\n",
		entry.Time.Format("2006-01-02 15:04:05"),
		strings.ToUpper(entry.Level.String()),
		file,
		line,
		entry.Message,
	)
	return []byte(format), nil
}

func trimCallerPath(path string) string {
	path = filepath.ToSlash(path)
	for _, root := range callerPathRoots {
		prefix := root + "/"
		if strings.HasPrefix(path, prefix) {
			return path
		}
	}
	best := len(path)
	for _, root := range callerPathRoots {
		marker := "/" + root + "/"
		if idx := strings.LastIndex(path, marker); idx >= 0 && idx+1 < best {
			best = idx + 1
		}
	}
	if best < len(path) {
		return path[best:]
	}
	return path
}
