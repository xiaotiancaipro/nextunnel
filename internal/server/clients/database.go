package clients

import (
	"fmt"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/migrations"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type database struct {
	config *configs.Database
	logger *zap.Logger
}

func NewDB(config *configs.Database, logger *zap.Logger) (*gorm.DB, error) {

	d := &database{
		config: config,
		logger: logger,
	}

	db, err := d.connect()
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %v", err)
	}

	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("database migration failed, %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL.DB: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil

}

func (d *database) migrate() error {
	db, err := d.connect()
	if err != nil {
		return fmt.Errorf("database connection failed: %v", err)
	}
	if err := migrations.Auto(db); err != nil {
		return fmt.Errorf("database migration failed: %v", err)
	}
	return nil
}

func (d *database) connect() (*gorm.DB, error) {
	if d.logger == nil {
		return nil, fmt.Errorf("database logger is required")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s timezone=UTC",
		d.config.Host,
		d.config.Port,
		d.config.Username,
		d.config.Password,
		d.config.Database,
		d.config.SSLModeOrDefault(),
	)
	conf := gorm.Config{
		Logger:  sharedlogger.NewGormLogger(d.logger, 0),
		NowFunc: func() time.Time { return sharedtimezone.NowUTC() },
	}
	return gorm.Open(postgres.Open(dsn), &conf)
}
