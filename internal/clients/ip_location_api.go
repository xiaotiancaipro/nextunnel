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

	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	"go.uber.org/zap"
)

const ipLocationAPIURL = "https://api.xiaotiancai.tech/api/v1/ip"

type IPLocationAPI struct {
	apiKey string
	client *http.Client
	logger *zap.Logger
}

type ipLocationAPIRequest struct {
	IP string `json:"ip"`
}

type ipLocationAPIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		Location struct {
			Country  string `json:"country"`
			Province string `json:"province"`
			City     string `json:"city"`
		} `json:"location"`
	} `json:"data"`
}

func NewIPLocationAPI(apiKey string, logger *zap.Logger) (*IPLocationAPI, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("ip_location api_key is required when type is api")
	}
	return &IPLocationAPI{
		apiKey: apiKey,
		client: &http.Client{Timeout: 5 * time.Second},
		logger: logger,
	}, nil
}

func (a *IPLocationAPI) Close() error {
	return nil
}

func (a *IPLocationAPI) Lookup(ipStr string) IPLocationResult {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return IPLocationResult{}
	}
	if utils.IsLocalIP(ipStr) {
		return IPLocationResult{}
	}

	body, err := json.Marshal(ipLocationAPIRequest{IP: ip.String()})
	if err != nil {
		a.logger.Warn(fmt.Sprintf("failed to encode ip location request: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}

	req, err := http.NewRequest(http.MethodPost, ipLocationAPIURL, bytes.NewReader(body))
	if err != nil {
		a.logger.Warn(fmt.Sprintf("failed to create ip location request: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		a.logger.Warn(fmt.Sprintf("failed to query ip location api: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		a.logger.Warn(fmt.Sprintf("failed to read ip location response: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}

	var parsed ipLocationAPIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		a.logger.Warn(fmt.Sprintf("failed to decode ip location response: ip=%s, err=%v", ipStr, err))
		return IPLocationResult{}
	}
	if parsed.Code != 200 || parsed.Data == nil {
		a.logger.Warn(fmt.Sprintf("ip location api returned error: ip=%s, code=%d, message=%s", ipStr, parsed.Code, parsed.Message))
		return IPLocationResult{}
	}

	return IPLocationResult{
		Country: parsed.Data.Location.Country,
		Region:  parsed.Data.Location.Province,
		City:    parsed.Data.Location.City,
	}
}
