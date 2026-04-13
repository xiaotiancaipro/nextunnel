package utils

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type Formatter struct {
	logrus.TextFormatter
	module string
}

func NewLogger(module string) *logrus.Logger {
	logger := logrus.New()
	formatter := Formatter{module: module}
	logger.SetFormatter(&formatter)
	logger.SetReportCaller(true)
	logger.SetLevel(logrus.InfoLevel)
	return logger
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	format := fmt.Sprintf(
		"%s - %s - %s - %s - %d - %s\n",
		entry.Time.Format("2006-01-02 15:04:05"),
		strings.ToUpper(entry.Level.String()),
		f.module,
		entry.Caller.Function,
		entry.Caller.Line,
		entry.Message,
	)
	return []byte(format), nil
}
