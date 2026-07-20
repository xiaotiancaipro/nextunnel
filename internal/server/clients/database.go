package clients

import (
	"fmt"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/clients/migrations"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	Config *configs.Database
	Logger *zap.Logger
	DB     *gorm.DB
}

func (c *Database) Init() error {
	db, err := c.connect()
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying SQL.DB: %w", err)
	}
	if err := migrations.Auto(db); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("database migration failed: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	c.DB = db
	return nil
}

func (c *Database) Close() error {
	if c == nil || c.DB == nil {
		return nil
	}
	sqlDB, err := c.DB.DB()
	c.DB = nil
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (c *Database) connect() (*gorm.DB, error) {
	if c.Logger == nil {
		return nil, fmt.Errorf("database logger is required")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s timezone=UTC",
		c.Config.Host,
		c.Config.Port,
		c.Config.Username,
		c.Config.Password,
		c.Config.Database,
		c.Config.SSLModeOrDefault(),
	)
	conf := gorm.Config{
		Logger:  sharedlogger.NewGormLogger(c.Logger, 0),
		NowFunc: func() time.Time { return sharedtimezone.NowUTC() },
	}
	return gorm.Open(postgres.Open(dsn), &conf)
}
