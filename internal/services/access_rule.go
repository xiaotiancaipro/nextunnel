package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"gorm.io/gorm"
)

type AccessRule struct {
	DB *gorm.DB
}

type RuleTarget struct {
	Ip       *string
	Country  *string
	Region   *string
	City     *string
	Category *string
}

func (r *AccessRule) NewRuleTarget(field, value string) (RuleTarget, error) {
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

func (r *AccessRule) NewCategoryRuleTarget(category string) (RuleTarget, error) {
	category, err := r.normalizeCategory(category)
	if err != nil {
		return RuleTarget{}, err
	}
	return RuleTarget{Category: &category}, nil
}

func (r *AccessRule) UpsertRule(target RuleTarget, status int16) error {
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
		if target.Category != nil {
			q = q.Where("category = ?", *target.Category)
		} else {
			q = q.Where("category IS NULL")
		}

		var record models.AccessRule
		err := q.First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&models.AccessRule{
				Ip:       target.Ip,
				Country:  target.Country,
				Region:   target.Region,
				City:     target.City,
				Category: target.Category,
				Status:   status,
			}).Error
		}
		if err != nil {
			return fmt.Errorf("failed to query access_rules: %w", err)
		}
		return tx.Model(&record).Update("status", status).Error
	})
}

func (r *AccessRule) IsAllowed(ip, country, region, city string, isLocal bool) (bool, error) {

	var rules []models.AccessRule
	if err := r.DB.Where("is_delete = ?", false).Find(&rules).Error; err != nil {
		return false, fmt.Errorf("failed to query access_rules: %w", err)
	}

	var best *models.AccessRule
	for i := range rules {
		rule := &rules[i]
		if !r.ruleMatches(*rule, ip, country, region, city, isLocal) {
			continue
		}
		if best == nil || r.isHigherPriorityRule(*rule, *best) {
			best = rule
		}
	}
	if best == nil {
		return true, nil
	}
	return best.Status == 1, nil

}

func (r *AccessRule) isHigherPriorityRule(candidate, current models.AccessRule) bool {
	candidateScore := r.ruleSpecificity(candidate)
	currentScore := r.ruleSpecificity(current)
	if candidateScore != currentScore {
		return candidateScore > currentScore
	}
	// Same specificity: Allow > Block
	if candidate.Status == 1 && current.Status == 0 {
		return true
	}
	return false
}

func (r *AccessRule) validateRuleTarget(target RuleTarget) error {
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
	if target.Category != nil {
		set++
	}
	if set == 0 {
		return fmt.Errorf("at least one of ip, country, region, city, category must be set")
	}
	return nil
}

func (r *AccessRule) ruleMatches(rule models.AccessRule, ip, country, region, city string, isLocal bool) bool {
	if !r.categoryMatches(rule.Category, isLocal) {
		return false
	}
	hasGeo := rule.Ip != nil || rule.Country != nil || rule.Region != nil || rule.City != nil
	if !hasGeo {
		return rule.Category != nil
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

func (r *AccessRule) categoryMatches(category *string, isLocal bool) bool {
	if category == nil {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(*category)) {
	case models.AccessRuleCategoryAll:
		return true
	case models.AccessRuleCategoryLocal:
		return isLocal
	case models.AccessRuleCategoryRemote:
		return !isLocal
	default:
		return false
	}
}

func (r *AccessRule) normalizeCategory(category string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(category)) {
	case models.AccessRuleCategoryAll:
		return models.AccessRuleCategoryAll, nil
	case models.AccessRuleCategoryLocal:
		return models.AccessRuleCategoryLocal, nil
	case models.AccessRuleCategoryRemote:
		return models.AccessRuleCategoryRemote, nil
	default:
		return "", fmt.Errorf("category must be ALL, LOCAL or REMOTE")
	}
}

func (r *AccessRule) ruleSpecificity(rule models.AccessRule) int {
	// Priority: IP > City > Region > Country > Category global rule
	if rule.Ip != nil {
		return 16
	}
	if rule.City != nil {
		return 8
	}
	if rule.Region != nil {
		return 4
	}
	if rule.Country != nil {
		return 2
	}
	if rule.Category != nil {
		switch strings.ToUpper(strings.TrimSpace(*rule.Category)) {
		case models.AccessRuleCategoryLocal, models.AccessRuleCategoryRemote:
			return 1
		case models.AccessRuleCategoryAll:
			return 0
		}
	}
	return -1
}
