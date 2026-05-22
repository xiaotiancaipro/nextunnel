package clients

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

var geoIPLocales = []string{
	"zh-CN",
	"zh",
	"en",
	"ja",
	"ko",
	"de",
	"fr",
	"es",
	"pt-BR",
	"ru",
	"it",
	"nl",
	"pl",
	"tr",
	"vi",
	"th",
	"id",
}

type GeoIP struct {
	reader *geoip2.Reader
}

type GeoIPResult struct {
	Country string
	Region  string
	City    string
}

func NewGeoIP(dbPath string) (*GeoIP, error) {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return nil, fmt.Errorf("dbPath is empty")
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("geoip database not found: %s", dbPath)
	}
	reader, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open geoip database: %w", err)
	}
	return &GeoIP{reader: reader}, nil
}

func (g *GeoIP) Close() error {
	if g == nil || g.reader == nil {
		return nil
	}
	return g.reader.Close()
}

func (g *GeoIP) Lookup(ipStr string) GeoIPResult {
	if g == nil || g.reader == nil {
		return GeoIPResult{}
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoIPResult{}
	}
	record, err := g.reader.City(ip)
	if err != nil {
		return GeoIPResult{}
	}
	var region string
	if n := len(record.Subdivisions); n > 0 {
		subdivision := record.Subdivisions[n-1]
		region = g.localizedName(subdivision.Names, subdivision.IsoCode)
	}
	countryCode := strings.ToUpper(strings.TrimSpace(record.Country.IsoCode))
	return GeoIPResult{
		Country: g.localizedName(record.Country.Names, countryCode),
		Region:  region,
		City:    g.localizedName(record.City.Names),
	}
}

func (g *GeoIP) localizedName(names map[string]string, fallbacks ...string) string {
	for _, locale := range geoIPLocales {
		if name := strings.TrimSpace(names[locale]); name != "" {
			return name
		}
	}
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			return name
		}
	}
	for _, fallback := range fallbacks {
		if name := strings.TrimSpace(fallback); name != "" {
			return name
		}
	}
	return ""
}
