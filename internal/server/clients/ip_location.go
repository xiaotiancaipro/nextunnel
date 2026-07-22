package clients

import (
	"context"
	"fmt"
	"net"
	"strings"

	xtcclient "github.com/xiaotiancai-tech/sdk-go/client"
	xtcservice "github.com/xiaotiancai-tech/sdk-go/service"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	sharednetwork "github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	"go.uber.org/zap"
)

type IPLocation struct {
	Config            *configs.IPLocation
	Logger            *zap.Logger
	client            *xtcclient.Client
	ipLocationService *xtcservice.IPLocation
}

type IPLocationResult struct {
	Country string
	Region  string
	City    string
}

func (c *IPLocation) Init() error {
	client, err := xtcclient.New(xtcclient.WithAPIKey(c.Config.APIKey))
	if err != nil {
		return fmt.Errorf("new xtc client failed: %w", err)
	}
	c.client = client
	c.ipLocationService = xtcservice.NewIPLocation(client)
	return nil
}

func (c *IPLocation) Close() error {
	return c.client.Close()
}

func (c *IPLocation) Lookup(ipStr string) IPLocationResult {

	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return IPLocationResult{}
	}

	if sharednetwork.IsLocalIP(ipStr) {
		return IPLocationResult{}
	}

	lookup, err := c.ipLocationService.Lookup(context.Background(), ipStr)
	if err != nil {
		c.Logger.Error(fmt.Sprintf("ip location returned error: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}

	return IPLocationResult{
		Country: lookup.Location.Country,
		Region:  lookup.Location.Province,
		City:    lookup.Location.City,
	}

}
