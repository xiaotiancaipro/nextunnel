package models

import (
	"time"

	"github.com/google/uuid"
)

const ClientCertTable = "client_cert"

type ClientCert struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;"`          // Primary key UUID
	ClientId  uuid.UUID `gorm:"type:uuid;not null"`                                        // Owning tunnel client UUID
	CertPath  string    `gorm:"type:text;not null"`                                        // Certificate storage path
	ExpiredAt time.Time `gorm:"type:timestamptz;not null"`                                 // Certificate expiration time (UTC)
	IsDelete  bool      `gorm:"type:boolean;default:0;not null;"`                          // Soft delete flag
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record creation timestamp (UTC)
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"` // Record last update timestamp (UTC)
}

func (ClientCert) TableName() string {
	return ClientCertTable
}
