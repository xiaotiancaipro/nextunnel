package services

import (
	"fmt"

	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	sharedstring "github.com/xiaotiancaipro/nextunnel/internal/shared/string"
	"gorm.io/gorm"
)

type AccessLog struct {
	DB                 *gorm.DB
	ClientService      *Client
	ClientProxyService *ClientProxy
}

func (s *AccessLog) Record(clientId, proxyName, ip, country, region, city string, isLocal bool, status int16) error {
	clientUUID, err := s.ClientService.ResolveClientId(s.DB, clientId)
	if err != nil {
		return fmt.Errorf("resolve client_id: %w", err)
	}
	proxyUUID, err := s.ClientProxyService.ResolveProxyId(s.DB, clientUUID, proxyName)
	if err != nil {
		return fmt.Errorf("resolve proxy_id: %w", err)
	}
	return s.DB.Model(&models.AccessLog{}).Create(map[string]any{
		"ClientId": clientUUID,
		"ProxyId":  proxyUUID,
		"Ip":       ip,
		"Category": s.categoryFromIP(isLocal),
		"Country":  sharedstring.NullIfEmpty(country),
		"Region":   sharedstring.NullIfEmpty(region),
		"City":     sharedstring.NullIfEmpty(city),
		"Status":   status,
	}).Error
}

func (s *AccessLog) categoryFromIP(isLocal bool) string {
	if isLocal {
		return models.AccessLogCategoryLocal
	}
	return models.AccessLogCategoryRemote
}
