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

// AccessRule defines IP/geo/category-based allow or block rules
type AccessRule struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;"`          // Primary key UUID
	Ip        *string   `gorm:"type:varchar(128);default:null;"`                           // Target IP address for matching; null matches any IP
	City      *string   `gorm:"type:varchar(256);default:null;"`                           // Target city for matching; null matches any city
	Region    *string   `gorm:"type:varchar(256);default:null;"`                           // Target region for matching; null matches any region
	Country   *string   `gorm:"type:varchar(256);default:null;"`                           // Target country for matching; null matches any country
	Category  *string   `gorm:"type:varchar(128);default:null;"`                           // Traffic category: ALL, LOCAL, or REMOTE
	Status    int16     `gorm:"type:smallint;not null;"`                                   // Rule action: 0 blocked, 1 allowed
	IsDelete  bool      `gorm:"type:boolean;default:0;not null;"`                          // Soft delete flag
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record creation timestamp (UTC)
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record last update timestamp (UTC)
}

func (AccessRule) TableName() string {
	return AccessRuleTable
}
