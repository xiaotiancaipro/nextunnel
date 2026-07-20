package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	sharednetwork "github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	"go.uber.org/zap"
)

const ipLocationAPIURL = "https://api.xiaotiancai.tech/api/v1/ip"

type IPLocation struct {
	Config *configs.IPLocation
	Logger *zap.Logger
	client *http.Client
}

type IPLocationResult struct {
	Country string
	Region  string
	City    string
}

type apiRequest struct {
	IP string `json:"ip"`
}

type apiResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *apiResponseData `json:"data"`
}

type apiResponseData struct {
	Location apiResponseDataLocation `json:"location"`
}

type apiResponseDataLocation struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
}

func (c *IPLocation) Init() error {
	c.client = &http.Client{Timeout: 5 * time.Second}
	return nil
}

func (c *IPLocation) Lookup(ipStr string) IPLocationResult {

	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return IPLocationResult{}
	}
	if sharednetwork.IsLocalIP(ipStr) {
		return IPLocationResult{}
	}

	body, err := json.Marshal(apiRequest{IP: ip.String()})
	if err != nil {
		c.Logger.Warn(fmt.Sprintf("failed to encode ip location request: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}

	req, err := http.NewRequest(http.MethodPost, ipLocationAPIURL, bytes.NewReader(body))
	if err != nil {
		c.Logger.Warn(fmt.Sprintf("failed to create ip location request: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Logger.Warn(fmt.Sprintf("failed to query ip location api: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		c.Logger.Warn(fmt.Sprintf("failed to read ip location response: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}

	var parsed apiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		c.Logger.Warn(fmt.Sprintf("failed to decode ip location response: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}
	if parsed.Code != 200 || parsed.Data == nil {
		c.Logger.Warn(fmt.Sprintf("ip location api returned error: ip=%s, code=%d, message=%s", ipStr, parsed.Code, parsed.Message))
		return IPLocationResult{}
	}

	return IPLocationResult{
		Country: parsed.Data.Location.Country,
		Region:  parsed.Data.Location.Province,
		City:    parsed.Data.Location.City,
	}

}

func (c *IPLocation) Close() error {
	return nil
}
