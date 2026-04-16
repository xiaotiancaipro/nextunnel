package logger

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type Formatter struct {
	logrus.TextFormatter
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	format := fmt.Sprintf(
		"%s - %s - %s - %d - %s\n",
		entry.Time.Format("2006-01-02 15:04:05"),
		strings.ToUpper(entry.Level.String()),
		strings.Replace(entry.Caller.Function, "github.com/xiaotiancaipro/nextunnel/", "", 1),
		entry.Caller.Line,
		entry.Message,
	)
	return []byte(format), nil
}
