package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedhttp "github.com/xiaotiancaipro/nextunnel/internal/shared/http"
	sharednetwork "github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

type IPFilter struct {
	AccessRuleService *services.AccessRule
}

type ipFilterResponse struct {
	ID        string  `json:"id"`
	Status    int16   `json:"status"`
	Field     string  `json:"field"`
	Value     *string `json:"value,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

type ipFilterMutateRequest struct {
	Status int16  `json:"status"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

func (c_ *IPFilter) List(c *gin.Context) {
	rules, err := c_.AccessRuleService.ListRules()
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]ipFilterResponse, 0, len(rules))
	for i := range rules {
		items = append(items, c_.toIPFilterResponse(rules[i]))
	}
	sharedhttp.Response(c, http.StatusOK, gin.H{"items": items})
}

func (c_ *IPFilter) Upsert(c *gin.Context) {
	var req ipFilterMutateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != 0 && req.Status != 1 {
		sharedhttp.ResponseError(c, http.StatusBadRequest, "status must be 0 (block) or 1 (allow)")
		return
	}
	target, err := c_.buildRuleTarget(req.Field, req.Value)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := c_.AccessRuleService.UpsertRule(target, req.Status); err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(c, http.StatusOK, gin.H{"message": "ok"})
}

func (c_ *IPFilter) Delete(c *gin.Context) {
	var req ipFilterMutateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != 0 && req.Status != 1 {
		sharedhttp.ResponseError(c, http.StatusBadRequest, "status must be 0 (block) or 1 (allow)")
		return
	}
	target, err := c_.buildRuleTarget(req.Field, req.Value)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := c_.AccessRuleService.DeleteRule(target, req.Status); err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(c, http.StatusOK, gin.H{"message": "ok"})
}

func (c_ *IPFilter) buildRuleTarget(field, value string) (services.RuleTarget, error) {
	field = strings.TrimSpace(field)
	switch strings.ToUpper(field) {
	case "ALL", "LOCAL", "REMOTE":
		return c_.AccessRuleService.NewCategoryRuleTarget(field)
	case "IP":
		ip, err := sharednetwork.NormalizeIP(value)
		if err != nil {
			return services.RuleTarget{}, err
		}
		return c_.AccessRuleService.NewRuleTarget("ip", *ip)
	case "COUNTRY", "REGION", "CITY":
		return c_.AccessRuleService.NewRuleTarget(strings.ToLower(field), value)
	default:
		return services.RuleTarget{}, fmt.Errorf("unsupported field: " + field)
	}
}

func (c_ *IPFilter) toIPFilterResponse(rule models.AccessRule) ipFilterResponse {
	resp := ipFilterResponse{
		ID:        rule.Id.String(),
		Status:    rule.Status,
		CreatedAt: sharedtimezone.FormatUTC(rule.CreatedAt),
	}
	switch {
	case rule.Category != nil:
		resp.Field = "category"
		resp.Value = rule.Category
	case rule.Ip != nil:
		resp.Field = "ip"
		resp.Value = rule.Ip
	case rule.Country != nil:
		resp.Field = "country"
		resp.Value = rule.Country
	case rule.Region != nil:
		resp.Field = "region"
		resp.Value = rule.Region
	case rule.City != nil:
		resp.Field = "city"
		resp.Value = rule.City
	}
	return resp
}
