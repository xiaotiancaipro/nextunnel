package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(config *configs.Logs) (*zap.Logger, error) {

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		ConsoleSeparator: " - ",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	dailyRotate := &lumberjack.Logger{
		Filename:   config.File,
		MaxSize:    100,
		MaxBackups: 30,
		MaxAge:     7,
		Compress:   false,
		LocalTime:  true,
	}

	writeSyncer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(dailyRotate),
		zapcore.AddSync(os.Stdout),
	)

	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s', error: %v", config.Level, err)
	}

	go scheduleDailyLogRotation(dailyRotate)

	core := zapcore.NewCore(encoder, writeSyncer, level)
	return zap.New(core, zap.AddCaller()), nil

}

func scheduleDailyLogRotation(logger *lumberjack.Logger) {
	loc := time.Local
	for {
		now := time.Now().In(loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
		time.Sleep(time.Until(next))
		if err := logger.Rotate(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "logger: daily rotate: %v\n", err)
		}
	}
}
