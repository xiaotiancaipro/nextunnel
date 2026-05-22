package services

import (
	"errors"
	"fmt"

	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"gorm.io/gorm"
)

type IpAddress struct {
	DB *gorm.DB
}

func (i *IpAddress) UpsertIPStatus(ip string, status int16) error {
	return i.DB.Transaction(func(tx *gorm.DB) error {
		var record models.IpAddress
		err := tx.Where("is_delete = ? AND ip = ?", false, ip).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Model(&models.IpAddress{}).Create(map[string]any{
				"Ip":     ip,
				"Count":  uint64(0),
				"Status": status,
			}).Error
		}
		if err != nil {
			return fmt.Errorf("failed to query ip address: %w", err)
		}
		return tx.Model(&record).Update("status", status).Error
	})
}
