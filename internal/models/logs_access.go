package models

import (
	"time"

	"github.com/google/uuid"
)

const LogsAccessTable = "logs_access"

const (
	LogsAccessCategoryLocal  = "LOCAL"
	LogsAccessCategoryRemote = "REMOTE"
)

type LogsAccess struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Ip        string    `gorm:"type:string;not null"`
	Category  *string   `gorm:"type:varchar(256);not null"` // LOCAL, REMOTE
	Country   *string   `gorm:"type:varchar(256);default:null"`
	Region    *string   `gorm:"type:varchar(256);default:null"`
	City      *string   `gorm:"type:varchar(256);default:null"`
	Status    int16     `gorm:"type:smallint;not null"` // 0 is blocked, 1 is allowed
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
}

func (LogsAccess) TableName() string {
	return LogsAccessTable
}
