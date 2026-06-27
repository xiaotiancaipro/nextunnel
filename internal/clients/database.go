package clients

import (
	"fmt"
	"time"

	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/timezone"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var tables = map[string]any{
	models.ClientTable:     models.Client{},
	models.ProxyTable:      models.Proxy{},
	models.AccessLogTable:  models.AccessLog{},
	models.AccessRuleTable: models.AccessRule{},
}

type database struct {
	config *configs.Database
	tables map[string]any
	logger *zap.Logger
}

func NewDB(config *configs.Database, logger *zap.Logger) (*gorm.DB, error) {

	d := &database{
		config: config,
		tables: tables,
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
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return fmt.Errorf("failed to enable uuid-ossp extension: %v", err)
	}
	for name, table := range d.tables {
		if err_ := db.AutoMigrate(&table); err_ != nil {
			return fmt.Errorf("table migration failed, TableName=%s: %v", name, err_)
		}
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
		Logger:  logger.NewGormLogger(d.logger, 0),
		NowFunc: func() time.Time { return timezone.NowUTC() },
	}
	return gorm.Open(postgres.Open(dsn), &conf)
}
