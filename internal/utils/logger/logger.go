package logger

import (
	"github.com/sirupsen/logrus"
)

func New(module string) *logrus.Logger {
	logger := logrus.New()
	formatter := Formatter{module: module}
	logger.SetFormatter(&formatter)
	logger.SetReportCaller(true)
	logger.SetLevel(logrus.InfoLevel)
	return logger
}
