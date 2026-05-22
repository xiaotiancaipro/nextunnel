package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"gorm.io/gorm"
)

type RulesIp struct {
	DB *gorm.DB
}

func (r *RulesIp) UpsertIPRule(ip string, status int16) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var record models.RulesIp
		err := tx.Where("is_delete = ? AND ip = ?", false, ip).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Model(&models.RulesIp{}).Create(map[string]any{
				"Ip":     new(ip),
				"Status": status,
			}).Error
		}
		if err != nil {
			return fmt.Errorf("failed to query rules_ip: %w", err)
		}
		return tx.Model(&record).Update("status", status).Error
	})
}

func (r *RulesIp) IsAllowed(ip, country, region, city string) (bool, error) {

	var rules []models.RulesIp
	if err := r.DB.Where("is_delete = ?", false).Find(&rules).Error; err != nil {
		return false, fmt.Errorf("failed to query rules_ip: %w", err)
	}

	var best *models.RulesIp
	bestScore := -1
	for i := range rules {
		rule := &rules[i]
		if !r.ruleMatches(*rule, ip, country, region, city) {
			continue
		}
		score := r.ruleSpecificity(*rule)
		if score > bestScore {
			bestScore = score
			best = rule
			continue
		}
		if score == bestScore && best != nil && rule.Status == 0 {
			best = rule
		}
	}
	if best == nil {
		return true, nil
	}
	return best.Status == 1, nil

}

func (r *RulesIp) ruleMatches(rule models.RulesIp, ip, country, region, city string) bool {
	if rule.Ip == nil && rule.Country == nil && rule.Region == nil && rule.City == nil {
		return false
	}
	if rule.Ip != nil && strings.TrimSpace(*rule.Ip) != ip {
		return false
	}
	if rule.Country != nil && strings.TrimSpace(*rule.Country) != country {
		return false
	}
	if rule.Region != nil && strings.TrimSpace(*rule.Region) != region {
		return false
	}
	if rule.City != nil && strings.TrimSpace(*rule.City) != city {
		return false
	}
	return true
}

func (r *RulesIp) ruleSpecificity(rule models.RulesIp) int {
	score := 0
	if rule.Ip != nil {
		score += 8
	}
	if rule.City != nil {
		score += 4
	}
	if rule.Region != nil {
		score += 2
	}
	if rule.Country != nil {
		score += 1
	}
	return score
}
