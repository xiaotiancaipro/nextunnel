package services

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"gorm.io/gorm"
)

type ClientProxy struct {
	Database *clients.Database
}

func (s *ClientProxy) SyncFromApply(clientId uuid.UUID, desired map[string]sharedprotocol.ProxiesApplyMsgItem) error {
	var existing []models.ClientProxy
	if err := s.Database.DB.Where("client_id = ?", clientId).Find(&existing).Error; err != nil {
		return fmt.Errorf("failed to query proxies: %w", err)
	}

	desiredNames := make(map[string]struct{}, len(desired))
	for name, proxy := range desired {
		desiredNames[name] = struct{}{}
		if err := s.upsert(clientId, name, proxy); err != nil {
			return err
		}
	}

	for _, row := range existing {
		if _, ok := desiredNames[row.Name]; ok {
			continue
		}
		if err := s.Database.DB.Model(&row).Update("status", int16(0)).Error; err != nil {
			return fmt.Errorf("failed to mark proxy %q offline: %w", row.Name, err)
		}
	}
	return nil
}

func (s *ClientProxy) SetAllOffline(clientId uuid.UUID) error {
	if err := s.Database.DB.Model(&models.ClientProxy{}).Where("client_id = ?", clientId).Update("status", int16(0)).Error; err != nil {
		return fmt.Errorf("failed to mark client proxies offline: %w", err)
	}
	return nil
}

func (s *ClientProxy) ResolveProxyId(db *gorm.DB, clientId uuid.UUID, name string) (uuid.UUID, error) {
	var proxy models.ClientProxy
	if err := db.Where("client_id = ? AND name = ?", clientId, name).First(&proxy).Error; err != nil {
		return uuid.Nil, fmt.Errorf("proxy not found: %w", err)
	}
	return proxy.Id, nil
}

func (s *ClientProxy) upsert(clientId uuid.UUID, name string, proxy sharedprotocol.ProxiesApplyMsgItem) error {
	var row models.ClientProxy
	err := s.Database.DB.Where("client_id = ? AND name = ?", clientId, name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = models.ClientProxy{
			ClientId:  clientId,
			Name:      name,
			Type:      proxy.Type,
			Port:      strconv.Itoa(proxy.RemotePort),
			LocalIp:   proxy.LocalIP,
			LocalPort: strconv.Itoa(proxy.LocalPort),
			Status:    1,
		}
		if err := s.Database.DB.Create(&row).Error; err != nil {
			return fmt.Errorf("failed to create proxy %q: %w", name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to query proxy %q: %w", name, err)
	}

	updates := map[string]interface{}{
		"type":       proxy.Type,
		"port":       strconv.Itoa(proxy.RemotePort),
		"local_ip":   proxy.LocalIP,
		"local_port": strconv.Itoa(proxy.LocalPort),
		"status":     int16(1),
	}
	if err := s.Database.DB.Model(&row).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update proxy %q: %w", name, err)
	}
	return nil
}
