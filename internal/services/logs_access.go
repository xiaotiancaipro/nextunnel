package services

import (
	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	"gorm.io/gorm"
)

type LogsAccess struct {
	DB *gorm.DB
}

func (l *LogsAccess) Record(ip, country, region, city string, isLocal bool, status int16) error {
	return l.DB.Model(&models.LogsAccess{}).Create(map[string]any{
		"Ip":       ip,
		"Category": l.LogsAccessCategoryFromIP(isLocal),
		"Country":  utils.NullIfEmpty(country),
		"Region":   utils.NullIfEmpty(region),
		"City":     utils.NullIfEmpty(city),
		"Status":   status,
	}).Error
}

func (l *LogsAccess) LogsAccessCategoryFromIP(isLocal bool) string {
	if isLocal {
		return models.LogsAccessCategoryLocal
	}
	return models.LogsAccessCategoryRemote
}
