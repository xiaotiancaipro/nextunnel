package models

import (
	"time"

	"github.com/google/uuid"
)

const ClientTable = "client"

type Client struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;"`          // Primary key UUID
	Name      string    `gorm:"type:varchar(255);not null;unique;"`                        // Unique client identifier name
	PortStart int       `gorm:"default:null;"`                                             // Inclusive start of allocated remote port range
	PortEnd   int       `gorm:"default:null;"`                                             // Inclusive end of allocated remote port range
	IsDelete  bool      `gorm:"type:boolean;default:0;not null;"`                          // Soft delete flag
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record creation timestamp (UTC)
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record last update timestamp (UTC)
}

func (Client) TableName() string {
	return ClientTable
}
