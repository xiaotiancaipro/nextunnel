package services

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	sharednetwork "github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	"go.uber.org/zap"
)

const ruleCacheTTL = 10 * time.Second

type AccessFilter struct {
	Logger            *zap.Logger
	Database          *clients.Database
	IPLocation        *clients.IPLocation
	AccessRuleService *AccessRule
	AccessLogService  *AccessLog
	ruleCacheMu       sync.RWMutex
	ruleCache         []models.AccessRule
	ruleCacheAt       time.Time
}

func (s *AccessFilter) Check(addr net.Addr, clientID, proxyName string) (ip, region string, err error) {
	host := addr.String()
	if parsedHost, _, splitErr := net.SplitHostPort(host); splitErr == nil {
		host = parsedHost
	}

	ipP, err := sharednetwork.NormalizeIP(host)
	if err != nil {
		return sharednetwork.UnknownIP, sharednetwork.UnknownIP, fmt.Errorf("failed to parse remote ip")
	}

	geo := s.IPLocation.Lookup(*ipP)
	region = s.formatRegion(geo.Country, geo.Region, geo.City)
	isLocal := sharednetwork.IsLocalIP(*ipP)

	rules, err := s.cachedRules()
	if err != nil {
		return *ipP, region, err
	}
	allowed := s.AccessRuleService.Evaluate(rules, *ipP, geo.Country, geo.Region, geo.City, isLocal)

	status := int16(0)
	if allowed {
		status = 1
	}
	if err := s.AccessLogService.Record(clientID, proxyName, *ipP, geo.Country, geo.Region, geo.City, isLocal, status); err != nil {
		s.Logger.Warn(fmt.Sprintf("failed to record access log: ip=%s, err=%v", *ipP, err))
	}

	if !allowed {
		return *ipP, region, fmt.Errorf("matched deny list")
	}
	return *ipP, region, nil
}

func (s *AccessFilter) cachedRules() ([]models.AccessRule, error) {
	s.ruleCacheMu.RLock()
	if s.ruleCache != nil && time.Since(s.ruleCacheAt) < ruleCacheTTL {
		rules := s.ruleCache
		s.ruleCacheMu.RUnlock()
		return rules, nil
	}
	s.ruleCacheMu.RUnlock()

	var rules []models.AccessRule
	if err := s.Database.DB.Where("is_delete = ?", false).Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to query access_rules: %w", err)
	}

	s.ruleCacheMu.Lock()
	s.ruleCache = rules
	s.ruleCacheAt = time.Now()
	s.ruleCacheMu.Unlock()

	return rules, nil
}

func (s *AccessFilter) formatRegion(country, region, city string) string {
	parts := make([]string, 0, 3)
	if country != "" {
		parts = append(parts, country)
	}
	if region != "" {
		parts = append(parts, region)
	}
	if city != "" {
		parts = append(parts, city)
	}
	if len(parts) == 0 {
		return sharednetwork.UnknownIP
	}
	return strings.Join(parts, "/")
}
