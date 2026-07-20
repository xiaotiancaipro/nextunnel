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

func (c_ *Client) List(c *gin.Context) {
	clients, err := c_.ClientService.List()
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]clientResponse, 0, len(clients))
	for i := range clients {
		items = append(items, c_.toClientResponse(clients[i]))
	}
	sharedhttp.Response(c, http.StatusOK, gin.H{"items": items})
}

func (c_ *Client) Create(c *gin.Context) {
	var req createClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	client, err := c_.ClientService.Create(req.Name, req.PortStart, req.PortEnd)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(c, http.StatusCreated, c_.toClientResponse(*client))
}

func (c_ *Client) Delete(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		sharedhttp.ResponseError(c, http.StatusBadRequest, "client name is required")
		return
	}
	client, err := c_.ClientService.GetByName(name)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	if err := c_.ClientService.Delete(name); err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := c_.ClientCertService.DeleteAllForClient(client.Id, name); err != nil && !os.IsNotExist(err) {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	sharedhttp.Response(c, http.StatusOK, gin.H{"message": "ok"})
}

func (c_ *Client) toClientResponse(client models.Client) clientResponse {
	return clientResponse{
		ID:        client.Id.String(),
		Name:      client.Name,
		PortStart: client.PortStart,
		PortEnd:   client.PortEnd,
		CreatedAt: sharedtimezone.FormatUTC(client.CreatedAt),
	}
}
