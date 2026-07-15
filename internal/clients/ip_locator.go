package clients

import (
	"fmt"
	"strings"

	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"go.uber.org/zap"
)

type IPLocationResult struct {
	Country string
	Region  string
	City    string
}

type IPLocator interface {
	Lookup(ipStr string) IPLocationResult
	Close() error
}

func NewIPLocator(cfg *configs.IPLocation, logger *zap.Logger) (IPLocator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ip_location config is required")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case "api":
		return NewIPLocationAPI(cfg.APIKey, logger)
	case "geoip", "":
		return NewGeoIP(cfg.GeoIPDbPath, cfg.GeoIPLocales)
	default:
		return nil, fmt.Errorf("unsupported ip_location type %q, use api or geoip", cfg.Type)
	}
}
