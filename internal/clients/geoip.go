package clients

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/oschwald/geoip2-golang"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
)

type GeoIP struct {
	reader  *geoip2.Reader
	locales []string
}

func NewGeoIP(dbPath string, locales []string) (*GeoIP, error) {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return nil, fmt.Errorf("geoip_db_path is empty")
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("geoip database not found: %s", dbPath)
	}
	reader, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open geoip database: %w", err)
	}
	return &GeoIP{reader: reader, locales: utils.Normalize(locales)}, nil
}

func (g *GeoIP) Close() error {
	if g == nil || g.reader == nil {
		return nil
	}
	return g.reader.Close()
}

func (g *GeoIP) Lookup(ipStr string) IPLocationResult {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return IPLocationResult{}
	}
	record, err := g.reader.City(ip)
	if err != nil {
		return IPLocationResult{}
	}
	var region string
	if n := len(record.Subdivisions); n > 0 {
		subdivision := record.Subdivisions[n-1]
		region = utils.PickName(subdivision.Names, g.locales)
	}
	return IPLocationResult{
		Country: utils.PickName(record.Country.Names, g.locales),
		Region:  region,
		City:    utils.PickName(record.City.Names, g.locales),
	}
}
