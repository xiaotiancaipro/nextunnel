package logger

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

type Formatter struct {
	logrus.TextFormatter
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	re := regexp.MustCompile(`^.*?(cmd/|internal/)`)
	format := fmt.Sprintf(
		"%s - %s - %s - %d - %s\n",
		entry.Time.Format("2006-01-02 15:04:05"),
		strings.ToUpper(entry.Level.String()),
		re.ReplaceAllString(entry.Caller.File, "$1"),
		entry.Caller.Line,
		entry.Message,
	)
	return []byte(format), nil
}
