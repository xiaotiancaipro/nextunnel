package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

const (
	maxSize      = 500 * 1024 * 1024
	rotationTime = 24 * time.Hour
	retentionAge = 30 * 24 * time.Hour
)

func New(module, file string) (*logrus.Logger, error) {

	logger := logrus.New()
	logger.SetFormatter(&Formatter{module: module})
	logger.SetReportCaller(true)
	logger.SetLevel(logrus.InfoLevel)

	if file == "" {
		logger.SetOutput(os.Stderr)
		return logger, nil
	}

	pattern, linkName := rotatePattern(file)
	writer, err := rotatelogs.New(
		pattern,
		rotatelogs.WithLinkName(linkName),
		rotatelogs.WithRotationTime(rotationTime),
		rotatelogs.WithRotationSize(maxSize),
		rotatelogs.WithMaxAge(retentionAge),
	)
	if err != nil {
		return nil, err
	}
	logger.SetOutput(io.MultiWriter(os.Stderr, writer))
	return logger, nil

}

func rotatePattern(file string) (pattern, linkName string) {
	file = filepath.Clean(file)
	dir := filepath.Dir(file)
	base := filepath.Base(file)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if ext == "" {
		return filepath.Join(dir, stem+".%Y%m%d"), file
	}
	return filepath.Join(dir, stem+".%Y%m%d"+ext), file
}
