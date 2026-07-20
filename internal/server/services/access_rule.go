package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	"gorm.io/gorm"
)

type AccessRule struct {
	Database *clients.Database
}

type RuleTarget struct {
	ip       *string
	country  *string
	region   *string
	city     *string
	category *string
}

func (s *AccessRule) NewRuleTarget(field, value string) (RuleTarget, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return RuleTarget{}, fmt.Errorf("%s cannot be empty", field)
	}
	target := RuleTarget{}
	switch field {
	case "ip":
		target.ip = &value
	case "country":
		target.country = &value
	case "region":
		target.region = &value
	case "city":
		target.city = &value
	default:
		return RuleTarget{}, fmt.Errorf("unsupported rule field: %s", field)
	}
	return target, nil
}

func (s *AccessRule) NewCategoryRuleTarget(category string) (RuleTarget, error) {
	category, err := s.normalizeCategory(category)
	if err != nil {
		return RuleTarget{}, err
	}
	return RuleTarget{category: &category}, nil
}

func (s *AccessRule) UpsertRule(target RuleTarget, status int16) error {
	if err := s.validateRuleTarget(target); err != nil {
		return err
	}
	return s.Database.DB.Transaction(func(tx *gorm.DB) error {
		var record models.AccessRule
		err := s.targetQuery(tx, target).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&models.AccessRule{
				Ip:       target.ip,
				Country:  target.country,
				Region:   target.region,
				City:     target.city,
				Category: target.category,
				Status:   status,
			}).Error
		}
		if err != nil {
			return fmt.Errorf("failed to query access_rules: %w", err)
		}
		return tx.Model(&record).Update("status", status).Error
	})
}

func (s *AccessRule) ListRules() ([]models.AccessRule, error) {
	var rules []models.AccessRule
	if err := s.Database.DB.Where("is_delete = ?", false).Order("status DESC, created_at ASC").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to query access_rules: %w", err)
	}
	return rules, nil
}

func (s *AccessRule) DeleteRule(target RuleTarget, status int16) error {
	if err := s.validateRuleTarget(target); err != nil {
		return err
	}
	return s.Database.DB.Transaction(func(tx *gorm.DB) error {
		var record models.AccessRule
		err := s.targetQuery(tx, target).Where("status = ?", status).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("rule not found")
		}
		if err != nil {
			return fmt.Errorf("failed to query access_rules: %w", err)
		}
		return tx.Model(&record).Update("is_delete", true).Error
	})
}

func (s *AccessRule) targetQuery(tx *gorm.DB, target RuleTarget) *gorm.DB {
	q := tx.Where("is_delete = ?", false)
	if target.ip != nil {
		q = q.Where("ip = ?", *target.ip)
	} else {
		q = q.Where("ip IS NULL")
	}
	if target.country != nil {
		q = q.Where("country = ?", *target.country)
	} else {
		q = q.Where("country IS NULL")
	}
	if target.region != nil {
		q = q.Where("region = ?", *target.region)
	} else {
		q = q.Where("region IS NULL")
	}
	if target.city != nil {
		q = q.Where("city = ?", *target.city)
	} else {
		q = q.Where("city IS NULL")
	}
	if target.category != nil {
		q = q.Where("category = ?", *target.category)
	} else {
		q = q.Where("category IS NULL")
	}
	return q
}

func (s *AccessRule) Evaluate(rules []models.AccessRule, ip, country, region, city string, isLocal bool) bool {
	var best *models.AccessRule
	for i := range rules {
		rule := &rules[i]
		if !s.ruleMatches(*rule, ip, country, region, city, isLocal) {
			continue
		}
		if best == nil || s.isHigherPriorityRule(*rule, *best) {
			best = rule
		}
	}
	if best == nil {
		return true
	}
	return best.Status == 1
}

func (s *AccessRule) isHigherPriorityRule(candidate, current models.AccessRule) bool {
	candidateScore := s.ruleSpecificity(candidate)
	currentScore := s.ruleSpecificity(current)
	if candidateScore != currentScore {
		return candidateScore > currentScore
	}
	// Same specificity: Allow > Block
	if candidate.Status == 1 && current.Status == 0 {
		return true
	}
	return false
}

func (s *AccessRule) validateRuleTarget(target RuleTarget) error {
	set := 0
	if target.ip != nil {
		set++
	}
	if target.country != nil {
		set++
	}
	if target.region != nil {
		set++
	}
	if target.city != nil {
		set++
	}
	if target.category != nil {
		set++
	}
	if set == 0 {
		return fmt.Errorf("at least one of ip, country, region, city, category must be set")
	}
	return nil
}

func (s *AccessRule) ruleMatches(rule models.AccessRule, ip, country, region, city string, isLocal bool) bool {
	if !s.categoryMatches(rule.Category, isLocal) {
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

func (s *AccessRule) categoryMatches(category *string, isLocal bool) bool {
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

func (s *AccessRule) normalizeCategory(category string) (string, error) {
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

func (s *AccessRule) ruleSpecificity(rule models.AccessRule) int {
	if rule.Ip != nil {
		return 1 << 4
	}
	if rule.City != nil {
		return 1 << 3
	}
	if rule.Region != nil {
		return 1 << 2
	}
	if rule.Country != nil {
		return 1 << 1
	}
	if rule.Category != nil {
		switch strings.ToUpper(strings.TrimSpace(*rule.Category)) {
		case models.AccessRuleCategoryLocal, models.AccessRuleCategoryRemote:
			return 1 << 0
		case models.AccessRuleCategoryAll:
			return 0
		}
	}
	return -1
}
