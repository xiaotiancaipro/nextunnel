package controllers

import (
	"archive/zip"
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	sharedhttp "github.com/xiaotiancaipro/nextunnel/internal/shared/http"
	sharedstring "github.com/xiaotiancaipro/nextunnel/internal/shared/string"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

type ClientCert struct {
	Config            *configs.Configs
	ClientService     *services.Client
	ClientCertService *services.ClientCert
}

type clientCertResponse struct {
	ID        string  `json:"id"`
	CreatedAt string  `json:"createdAt"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	Serial    string  `json:"serial"`
}

type createClientCertRequest struct {
	ExpiresAt *string `json:"expiresAt"`
}

func (c *ClientCert) List(ctx *gin.Context) {
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
	items, err := c.ClientCertService.List(client.Id)
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]clientCertResponse, 0, len(items))
	for i := range items {
		resp = append(resp, c.toClientCertResponse(items[i]))
	}
	sharedhttp.Response(ctx, http.StatusOK, gin.H{"items": resp})
}

func (c *ClientCert) Create(ctx *gin.Context) {

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

	var req createClientCertRequest
	if ctx.Request.ContentLength > 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			sharedhttp.ResponseError(ctx, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw != "" {
			parsed, err := sharedtimezone.ParseRFC3339(raw)
			if err != nil {
				sharedhttp.ResponseError(ctx, http.StatusBadRequest, "expiresAt must be RFC3339 timestamp")
				return
			}
			expiresAt = &parsed
		}
	}

	info, err := c.ClientCertService.Create(client, expiresAt)
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(ctx, http.StatusCreated, c.toClientCertResponse(info))
}

func (c *ClientCert) Delete(ctx *gin.Context) {

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

	certIDRaw := strings.TrimSpace(ctx.Param("id"))
	certID, err := sharedstring.ParseUUID(certIDRaw)
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.ClientCertService.Delete(client.Id, certID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			sharedhttp.ResponseError(ctx, http.StatusNotFound, err.Error())
			return
		}
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(ctx, http.StatusOK, gin.H{"message": "ok"})
}

func (c *ClientCert) Download(ctx *gin.Context) {

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

	certIDRaw := strings.TrimSpace(ctx.Param("id"))
	certID, err := sharedstring.ParseUUID(certIDRaw)
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	certPEM, keyPEM, err := c.ClientCertService.ReadFiles(client.Id, certID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sharedhttp.ResponseError(ctx, http.StatusNotFound, err.Error())
			return
		}
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if err := c.writeClientCertZip(ctx, name, certIDRaw, certPEM, keyPEM); err != nil {
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
	}
}

func (c *ClientCert) DownloadCA(ctx *gin.Context) {
	if err := sharedcerts.Ensure(c.Config.Cert.Dir, c.Config.Cert.Host); err != nil {
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	abs, err := filepath.Abs(c.Config.Cert.Dir)
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	caPEM, err := os.ReadFile(filepath.Join(abs, sharedcerts.FileCACert))
	if err != nil {
		sharedhttp.ResponseError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Header("Content-Disposition", `attachment; filename="ca.crt"`)
	ctx.Data(http.StatusOK, "application/x-pem-file", caPEM)
}

func (c *ClientCert) toClientCertResponse(info services.ClientCertView) clientCertResponse {
	resp := clientCertResponse{
		ID:        info.ID,
		CreatedAt: sharedtimezone.FormatUTC(info.CreatedAt),
		Serial:    info.Serial,
	}
	if info.ExpiresAt != nil {
		formatted := sharedtimezone.FormatUTC(*info.ExpiresAt)
		resp.ExpiresAt = &formatted
	}
	return resp
}

func (c *ClientCert) writeClientCertZip(ctx *gin.Context, clientName, certID string, certPEM, keyPEM []byte) error {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for fileName, content := range map[string][]byte{
		sharedcerts.FileClientCert: certPEM,
		sharedcerts.FileClientKey:  keyPEM,
	} {
		fw, err := zw.Create(fileName)
		if err != nil {
			return err
		}
		if _, err := fw.Write(content); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}

	ctx.Header("Content-Disposition", `attachment; filename="`+clientName+`-`+certID+`-sharedcerts.zip"`)
	ctx.Data(http.StatusOK, "application/zip", buf.Bytes())
	return nil
}
