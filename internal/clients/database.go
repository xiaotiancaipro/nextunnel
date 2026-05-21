package clients

import (
	"fmt"
	"time"

	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var tables = map[string]any{
	models.IpAddressTable: models.IpAddress{},
}

type Database struct {
	Config        *configs.Database
	Tables        map[string]any
	Logger        *zap.Logger
	SlowThreshold time.Duration
}

func InitDB(config *configs.Database, logger *zap.Logger) *Database {
	return &Database{
		Config: config,
		Tables: tables,
		Logger: logger,
	}
}

func (d *Database) New() (*gorm.DB, error) {

	db, err := d.connect()
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %v", err)
	}

	if err := d.Migrate(); err != nil {
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

func (d *Database) Migrate() error {
	db, err := d.connect()
	if err != nil {
		return fmt.Errorf("database connection failed: %v", err)
	}
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return fmt.Errorf("failed to enable uuid-ossp extension: %v", err)
	}
	for name, table := range d.Tables {
		if err_ := db.AutoMigrate(&table); err_ != nil {
			return fmt.Errorf("table migration failed, TableName=%s: %v", name, err_)
		}
	}
	return nil
}

func (d *Database) connect() (*gorm.DB, error) {
	if d.Logger == nil {
		return nil, fmt.Errorf("database logger is required")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		d.Config.Host,
		d.Config.Port,
		d.Config.Username,
		d.Config.Password,
		d.Config.Database,
	)
	conf := gorm.Config{
		Logger: logger.NewGormLoggerFormatted(d.Logger, d.SlowThreshold),
	}
	return gorm.Open(postgres.Open(dsn), &conf)
}
