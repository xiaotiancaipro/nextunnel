package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const moduleImportPath = "github.com/xiaotiancaipro/nextunnel-server"

var repoRootDir string

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
		EncodeCaller:   repoRelativeCallerEncoder,
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

func init() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	repoRootDir = findRepoRoot(filepath.Dir(file))
}

func findRepoRoot(dir string) string {
	for {
		st, err := os.Stat(filepath.Join(dir, "go.mod"))
		if err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func repoRelativeCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	if !caller.Defined {
		enc.AppendString("undefined")
		return
	}
	enc.AppendString(formatRepoRelativeCaller(caller.File, caller.Line))
}

func formatRepoRelativeCaller(file string, line int) string {
	if rel, ok := pathRelativeToRepoRoot(file); ok {
		return fmt.Sprintf("%s:%d", rel, line)
	}
	return zapcore.EntryCaller{Defined: true, File: file, Line: line}.TrimmedPath()
}

func pathRelativeToRepoRoot(file string) (string, bool) {
	mod := moduleImportPath + "/"
	if strings.HasPrefix(file, mod) {
		return filepath.ToSlash(strings.TrimPrefix(file, mod)), true
	}
	if i := strings.Index(file, mod); i >= 0 {
		return filepath.ToSlash(file[i+len(mod):]), true
	}
	if repoRootDir != "" && filepath.IsAbs(file) {
		rel, err := filepath.Rel(repoRootDir, file)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel), true
		}
	}
	return "", false
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
