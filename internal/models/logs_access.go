package models

import (
	"time"

	"github.com/google/uuid"
)

const LogsAccessTable = "logs_access"

type LogsAccess struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Ip        string    `gorm:"type:string;not null;"`
	Country   *string   `gorm:"type:varchar(256);default:null"`
	Region    *string   `gorm:"type:varchar(256);default:null"`
	City      *string   `gorm:"type:varchar(256);default:null"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
}

func (LogsAccess) TableName() string {
	return LogsAccessTable
}
