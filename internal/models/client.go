package models

import (
	"time"

	"github.com/google/uuid"
)

const ClientTable = "client"

type Client struct {
	Id uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;"` // Primary key UUID
	// TODO
	Status    int16     `gorm:"type:smallint;not null;"`                                   // Access decision: 0 blocked, 1 allowed
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record creation timestamp (UTC)
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record last update timestamp (UTC)
}

func (Client) TableName() string {
	return ClientTable
}
