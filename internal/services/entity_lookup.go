package services

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"gorm.io/gorm"
)

func resolveClientId(db *gorm.DB, id string) (uuid.UUID, error) {
	if uid, err := uuid.Parse(id); err == nil {
		var client models.Client
		if err := db.Where("id = ? AND is_delete = ?", uid, false).First(&client).Error; err != nil {
			return uuid.Nil, fmt.Errorf("client not found: %w", err)
		}
		return client.Id, nil
	}
	var client models.Client
	if err := db.Where("name = ? AND is_delete = ?", id, false).First(&client).Error; err != nil {
		return uuid.Nil, fmt.Errorf("client not found: %w", err)
	}
	return client.Id, nil
}

func resolveProxyId(db *gorm.DB, clientId uuid.UUID, name string) (uuid.UUID, error) {
	var proxy models.Proxy
	if err := db.Where("client_id = ? AND name = ?", clientId, name).First(&proxy).Error; err != nil {
		return uuid.Nil, fmt.Errorf("proxy not found: %w", err)
	}
	return proxy.Id, nil
}
