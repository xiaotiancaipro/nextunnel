package models

import (
	"time"

	"github.com/google/uuid"
)

const ProxyTable = "proxy"

type Proxy struct {
	Id        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;"`                       // Primary key UUID
	ClientId  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uk_proxy_client_name,priority:1"`         // Owning tunnel client UUID
	Name      string    `gorm:"type:varchar(255);not null;uniqueIndex:uk_proxy_client_name,priority:2"` // Proxy rule name, unique within the client
	Type      string    `gorm:"type:varchar(255);not null"`                                             // Proxy protocol type, e.g. tcp
	Port      string    `gorm:"type:varchar(255);not null"`                                             // Remote listening port exposed on the server
	LocalIp   string    `gorm:"type:varchar(255);not null"`                                             // Backend target IP on the client side
	LocalPort string    `gorm:"type:varchar(255);not null"`                                             // Backend target port on the client side
	Status    int16     `gorm:"type:smallint;not null;"`                                                // Access decision: 0 blocked, 1 allowed
	CreatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"`              // Record creation timestamp (UTC)
	UpdatedAt time.Time `gorm:"type:timestamptz;default:timezone('utc', now());not null;"`              // Record last update timestamp (UTC)
}

func (Proxy) TableName() string {
	return ProxyTable
}
