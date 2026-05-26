package logger

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultSlowThreshold = 200 * time.Millisecond

var sqlWhitespaceReplacer = strings.NewReplacer(
	"\t", " ",
	"\n", " ",
	"  ", " ",
	"   ", " ",
	"    ", " ",
)

type GormLoggerFormatted struct {
	logger        *zap.Logger
	slowThreshold time.Duration
}

func NewGormLoggerFormatted(logger *zap.Logger, slowThreshold time.Duration) *GormLoggerFormatted {
	if slowThreshold <= 0 {
		slowThreshold = defaultSlowThreshold
	}
	return &GormLoggerFormatted{
		logger:        logger.Named("gorm"),
		slowThreshold: slowThreshold,
	}
}

func (f *GormLoggerFormatted) LogMode(level logger.LogLevel) logger.Interface {
	if level <= logger.Silent {
		return &GormLoggerFormatted{logger: zap.NewNop(), slowThreshold: f.slowThreshold}
	}
	return f
}

func (f *GormLoggerFormatted) Info(context.Context, string, ...any) {}

func (f *GormLoggerFormatted) Warn(context.Context, string, ...any) {}

func (f *GormLoggerFormatted) Error(context.Context, string, ...any) {}

func (f *GormLoggerFormatted) Trace(
	_ context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {

	elapsed := time.Since(begin)
	if err == nil && elapsed <= f.slowThreshold {
		return
	}

	sql, rows := fc()
	sql = sqlWhitespaceReplacer.Replace(sql)
	sql = sqlWhitespaceReplacer.Replace(sql)

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		fields = append(fields, zap.Error(err))
		f.logger.Error("query failed", fields...)
		return
	}
	f.logger.Warn("slow query", fields...)

}
