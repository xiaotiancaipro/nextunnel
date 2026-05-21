package models

import (
	"time"

	"github.com/google/uuid"
)

const IpFilterTable = "ip_filter"

type IpFilter struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Ip        string    `gorm:"type:string;not null"`
	Status    bool      `gorm:"type:boolean;not null"` // 1 is allowed, 0 is blocked
	IsDelete  bool      `gorm:"type:boolean;not null"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
}

func (IpFilter) TableName() string {
	return IpFilterTable
}
