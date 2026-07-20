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

func (c_ *ClientCert) List(c *gin.Context) {
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
	items, err := c_.ClientCertService.List(client.Id)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]clientCertResponse, 0, len(items))
	for i := range items {
		resp = append(resp, c_.toClientCertResponse(items[i]))
	}
	sharedhttp.Response(c, http.StatusOK, gin.H{"items": resp})
}

func (c_ *ClientCert) Create(c *gin.Context) {

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

	var req createClientCertRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			sharedhttp.ResponseError(c, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw != "" {
			parsed, err := sharedtimezone.ParseRFC3339(raw)
			if err != nil {
				sharedhttp.ResponseError(c, http.StatusBadRequest, "expiresAt must be RFC3339 timestamp")
				return
			}
			expiresAt = &parsed
		}
	}

	info, err := c_.ClientCertService.Create(client, expiresAt)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(c, http.StatusCreated, c_.toClientCertResponse(info))
}

func (c_ *ClientCert) Delete(c *gin.Context) {

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

	certIDRaw := strings.TrimSpace(c.Param("id"))
	certID, err := sharedstring.ParseUUID(certIDRaw)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := c_.ClientCertService.Delete(client.Id, certID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			sharedhttp.ResponseError(c, http.StatusNotFound, err.Error())
			return
		}
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.Response(c, http.StatusOK, gin.H{"message": "ok"})
}

func (c_ *ClientCert) Download(c *gin.Context) {

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

	certIDRaw := strings.TrimSpace(c.Param("id"))
	certID, err := sharedstring.ParseUUID(certIDRaw)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	certPEM, keyPEM, err := c_.ClientCertService.ReadFiles(client.Id, certID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sharedhttp.ResponseError(c, http.StatusNotFound, err.Error())
			return
		}
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := c_.writeClientCertZip(c, name, certIDRaw, certPEM, keyPEM); err != nil {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
	}
}

func (c_ *ClientCert) DownloadCA(c *gin.Context) {
	if err := sharedcerts.Ensure(c_.Config.Cert.Dir, c_.Config.Cert.Host); err != nil {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	abs, err := filepath.Abs(c_.Config.Cert.Dir)
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	caPEM, err := os.ReadFile(filepath.Join(abs, sharedcerts.FileCACert))
	if err != nil {
		sharedhttp.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Disposition", `attachment; filename="ca.crt"`)
	c.Data(http.StatusOK, "application/x-pem-file", caPEM)
}

func (c_ *ClientCert) toClientCertResponse(info services.ClientCertView) clientCertResponse {
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

func (c_ *ClientCert) writeClientCertZip(c *gin.Context, clientName, certID string, certPEM, keyPEM []byte) error {
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

	c.Header("Content-Disposition", `attachment; filename="`+clientName+`-`+certID+`-sharedcerts.zip"`)
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
	return nil
}
