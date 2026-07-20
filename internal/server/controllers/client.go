package controllers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedhttp "github.com/xiaotiancaipro/nextunnel/internal/shared/http"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

type Client struct {
	ClientService     *services.Client
	ClientCertService *services.ClientCert
}

type clientResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PortStart int    `json:"portStart"`
	PortEnd   int    `json:"portEnd"`
	CreatedAt string `json:"createdAt"`
}

type createClientRequest struct {
	Name      string `json:"name"`
	PortStart int    `json:"portStart"`
	PortEnd   int    `json:"portEnd"`
}

func (c *Client) List(ctx *gin.Context) {
	clients, err := c.ClientService.List()
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]clientResponse, 0, len(clients))
	for i := range clients {
		items = append(items, c.toClientResponse(clients[i]))
	}
	sharedhttp.Response(ctx, http.StatusOK, gin.H{"items": items})
}

func (c *Client) Create(ctx *gin.Context) {
	var req createClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	client, err := c.ClientService.Create(req.Name, req.PortStart, req.PortEnd)
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(ctx, http.StatusCreated, c.toClientResponse(*client))
}

func (c *Client) Delete(ctx *gin.Context) {
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, "client name is required")
		return
	}
	client, err := c.ClientService.GetByName(name)
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusNotFound, err.Error())
		return
	}
	if err := c.ClientService.Delete(name); err != nil {
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.ClientCertService.DeleteAllForClient(client.Id, name); err != nil && !os.IsNotExist(err) {
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	sharedhttp.Response(ctx, http.StatusOK, gin.H{"message": "ok"})
}

func (c *Client) toClientResponse(client models.Client) clientResponse {
	return clientResponse{
		ID:        client.Id.String(),
		Name:      client.Name,
		PortStart: client.PortStart,
		PortEnd:   client.PortEnd,
		CreatedAt: sharedtimezone.FormatUTC(client.CreatedAt),
	}
}
