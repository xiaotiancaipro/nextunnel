package models

import (
	"time"

	"github.com/google/uuid"
)

const IpAddressTable = "ip_address"

type IpAddress struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Ip        string    `gorm:"type:string;not null;uniqueIndex"`
	Country   *string   `gorm:"type:varchar(256);default:null"`
	Region    *string   `gorm:"type:varchar(256);default:null"`
	City      *string   `gorm:"type:varchar(256);default:null"`
	Count     uint64    `gorm:"type:bigint;default:1;not null"`
	Status    int16     `gorm:"type:smallint;default:-1;not null"` // -1 is unknown state, 0 is blocked, 1 is allowed
	IsDelete  bool      `gorm:"type:boolean;default:0;not null"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
}

func (IpAddress) TableName() string {
	return IpAddressTable
}
