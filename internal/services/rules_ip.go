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

type RuleTarget struct {
	Ip      *string
	Country *string
	Region  *string
	City    *string
}

func (r *RulesIp) NewRuleTarget(field, value string) (RuleTarget, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return RuleTarget{}, fmt.Errorf("%s cannot be empty", field)
	}
	target := RuleTarget{}
	switch field {
	case "ip":
		target.Ip = &value
	case "country":
		target.Country = &value
	case "region":
		target.Region = &value
	case "city":
		target.City = &value
	default:
		return RuleTarget{}, fmt.Errorf("unsupported rule field: %s", field)
	}
	return target, nil
}

func (r *RulesIp) UpsertRule(target RuleTarget, status int16) error {
	if err := r.validateRuleTarget(target); err != nil {
		return err
	}
	return r.DB.Transaction(func(tx *gorm.DB) error {
		q := tx.Where("is_delete = ?", false)
		if target.Ip != nil {
			q = q.Where("ip = ?", *target.Ip)
		} else {
			q = q.Where("ip IS NULL")
		}
		if target.Country != nil {
			q = q.Where("country = ?", *target.Country)
		} else {
			q = q.Where("country IS NULL")
		}
		if target.Region != nil {
			q = q.Where("region = ?", *target.Region)
		} else {
			q = q.Where("region IS NULL")
		}
		if target.City != nil {
			q = q.Where("city = ?", *target.City)
		} else {
			q = q.Where("city IS NULL")
		}
		var record models.RulesIp
		err := q.First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Model(&models.RulesIp{}).Create(map[string]any{
				"Ip":      target.Ip,
				"Country": target.Country,
				"Region":  target.Region,
				"City":    target.City,
				"Status":  status,
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

func (r *RulesIp) validateRuleTarget(target RuleTarget) error {
	set := 0
	if target.Ip != nil {
		set++
	}
	if target.Country != nil {
		set++
	}
	if target.Region != nil {
		set++
	}
	if target.City != nil {
		set++
	}
	if set != 1 {
		return fmt.Errorf("exactly one of ip, country, region, city must be set")
	}
	return nil
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
