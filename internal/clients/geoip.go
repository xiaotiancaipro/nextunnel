package clients

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

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
		region = subdivision.Names["en"]
	}
	return GeoIPResult{
		Country: strings.ToUpper(strings.TrimSpace(record.Country.IsoCode)),
		Region:  region,
		City:    record.City.Names["en"],
	}
}
