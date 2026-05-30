package models

import (
	"time"

	"github.com/google/uuid"
)

const AccessLogTable = "access_log"

const (
	AccessLogCategoryLocal  = "LOCAL"
	AccessLogCategoryRemote = "REMOTE"
)

// AccessLog records each access attempt and its geo/decision metadata
type AccessLog struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;"`          // Primary key UUID
	Ip        string    `gorm:"type:varchar(128);not null;"`                               // Client IP address
	Category  *string   `gorm:"type:varchar(128);not null;"`                               // Access source category: LOCAL or REMOTE
	Country   *string   `gorm:"type:varchar(256);default:null;"`                           // GeoIP country name
	Region    *string   `gorm:"type:varchar(256);default:null;"`                           // GeoIP region or state name
	City      *string   `gorm:"type:varchar(256);default:null;"`                           // GeoIP city name
	Status    int16     `gorm:"type:smallint;not null;"`                                   // Access decision: 0 blocked, 1 allowed
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record creation timestamp (UTC)
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record last update timestamp (UTC)
}

func (AccessLog) TableName() string {
	return AccessLogTable
}
