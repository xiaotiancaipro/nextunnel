package models

import (
	"time"

	"github.com/google/uuid"
)

const AccessRuleTable = "access_rule"

const (
	AccessRuleCategoryAll    = "ALL"
	AccessRuleCategoryLocal  = "LOCAL"
	AccessRuleCategoryRemote = "REMOTE"
)

type AccessRule struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Ip        *string   `gorm:"type:string;default:null"`
	Country   *string   `gorm:"type:varchar(256);default:null"`
	Region    *string   `gorm:"type:varchar(256);default:null"`
	City      *string   `gorm:"type:varchar(256);default:null"`
	Category  *string   `gorm:"type:varchar(256);default:null"` // ALL, LOCAL, REMOTE
	Status    int16     `gorm:"type:smallint;not null"`         // 0 is blocked, 1 is allowed
	IsDelete  bool      `gorm:"type:boolean;default:0;not null"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null"`
}

func (AccessRule) TableName() string {
	return AccessRuleTable
}
