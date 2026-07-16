package services

import (
	"fmt"

	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	string2 "github.com/xiaotiancaipro/nextunnel/internal/shared/string"
	"gorm.io/gorm"
)

type accessLog struct {
	db *gorm.DB
}

func newAccessLog(db *gorm.DB) *accessLog {
	return &accessLog{db: db}
}

func (l *accessLog) record(clientId, proxyName, ip, country, region, city string, isLocal bool, status int16) error {
	clientUUID, err := resolveClientId(l.db, clientId)
	if err != nil {
		return fmt.Errorf("resolve client_id: %w", err)
	}
	proxyUUID, err := resolveProxyId(l.db, clientUUID, proxyName)
	if err != nil {
		return fmt.Errorf("resolve proxy_id: %w", err)
	}
	return l.db.Model(&models.AccessLog{}).Create(map[string]any{
		"ClientId": clientUUID,
		"ProxyId":  proxyUUID,
		"Ip":       ip,
		"Category": l.categoryFromIP(isLocal),
		"Country":  string2.NullIfEmpty(country),
		"Region":   string2.NullIfEmpty(region),
		"City":     string2.NullIfEmpty(city),
		"Status":   status,
	}).Error
}

func (l *accessLog) categoryFromIP(isLocal bool) string {
	if isLocal {
		return models.AccessLogCategoryLocal
	}
	return models.AccessLogCategoryRemote
}
