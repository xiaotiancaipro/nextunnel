package controllers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	sharednetwork "github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/version", s.handleVersion)
	mux.HandleFunc("GET /api/clients", s.handleListClients)
	mux.HandleFunc("POST /api/clients", s.handleCreateClient)
	mux.HandleFunc("DELETE /api/clients/{name}", s.handleDeleteClient)
	mux.HandleFunc("GET /api/clients/{name}/sharedcerts", s.handleListClientCerts)
	mux.HandleFunc("POST /api/clients/{name}/sharedcerts", s.handleCreateClientCert)
	mux.HandleFunc("GET /api/clients/{name}/sharedcerts/{id}/download", s.handleDownloadClientCert)
	mux.HandleFunc("DELETE /api/clients/{name}/sharedcerts/{id}", s.handleDeleteClientCert)
	mux.HandleFunc("GET /api/ca", s.handleDownloadCA)
	mux.HandleFunc("GET /api/ip-filters", s.handleListIPFilters)
	mux.HandleFunc("POST /api/ip-filters", s.handleUpsertIPFilter)
	mux.HandleFunc("DELETE /api/ip-filters", s.handleDeleteIPFilter)
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": s.version})
}

type clientResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PortStart int    `json:"portStart"`
	PortEnd   int    `json:"portEnd"`
	CreatedAt string `json:"createdAt"`
}

func toClientResponse(client models.Client) clientResponse {
	return clientResponse{
		ID:        client.Id.String(),
		Name:      client.Name,
		PortStart: client.PortStart,
		PortEnd:   client.PortEnd,
		CreatedAt: sharedtimezone.FormatUTC(client.CreatedAt),
	}
}

func (s *Server) handleListClients(w http.ResponseWriter, _ *http.Request) {
	clients, err := s.clientService.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]clientResponse, 0, len(clients))
	for i := range clients {
		items = append(items, toClientResponse(clients[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createClientRequest struct {
	Name      string `json:"name"`
	PortStart int    `json:"portStart"`
	PortEnd   int    `json:"portEnd"`
}

func (s *Server) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	var req createClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	client, err := s.clientService.Create(req.Name, req.PortStart, req.PortEnd)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toClientResponse(*client))
}

func (s *Server) handleDeleteClient(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	client, ok := s.requireClientByName(w, name)
	if !ok {
		return
	}
	if err := s.clientService.Delete(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.clientCertService.DeleteAllForClient(client.Id, name); err != nil && !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

type clientCertResponse struct {
	ID        string  `json:"id"`
	CreatedAt string  `json:"createdAt"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	Serial    string  `json:"serial"`
}

func toClientCertResponse(info services.ClientCertView) clientCertResponse {
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

func (s *Server) requireClientByName(w http.ResponseWriter, name string) (*models.Client, bool) {
	if name == "" {
		writeError(w, http.StatusBadRequest, "client name is required")
		return nil, false
	}
	client, err := s.clientService.GetByName(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return nil, false
	}
	return client, true
}

func (s *Server) handleListClientCerts(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	client, ok := s.requireClientByName(w, name)
	if !ok {
		return
	}

	items, err := s.clientCertService.List(client.Id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]clientCertResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toClientCertResponse(items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

type createClientCertRequest struct {
	ExpiresAt *string `json:"expiresAt"`
}

func (s *Server) handleCreateClientCert(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	client, ok := s.requireClientByName(w, name)
	if !ok {
		return
	}

	var req createClientCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw != "" {
			parsed, err := sharedtimezone.ParseRFC3339(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "expiresAt must be RFC3339 timestamp")
				return
			}
			expiresAt = &parsed
		}
	}

	info, err := s.clientCertService.Create(client, expiresAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toClientCertResponse(info))
}

func (s *Server) handleDeleteClientCert(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	certIDRaw := strings.TrimSpace(r.PathValue("id"))
	client, ok := s.requireClientByName(w, name)
	if !ok {
		return
	}
	certID, err := services.ParseCertID(certIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.clientCertService.Delete(client.Id, certID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func writeClientCertZip(w http.ResponseWriter, clientName, certID string, certPEM, keyPEM []byte) error {
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

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+clientName+`-`+certID+`-sharedcerts.zip"`)
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	return err
}

func (s *Server) handleDownloadClientCert(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	certIDRaw := strings.TrimSpace(r.PathValue("id"))
	client, ok := s.requireClientByName(w, name)
	if !ok {
		return
	}
	certID, err := services.ParseCertID(certIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	certPEM, keyPEM, err := s.clientCertService.ReadFiles(client.Id, certID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := writeClientCertZip(w, name, certIDRaw, certPEM, keyPEM); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) handleDownloadCA(w http.ResponseWriter, _ *http.Request) {
	if err := sharedcerts.Ensure(s.cfg.Cert.Dir, s.cfg.Cert.Host); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	abs, err := filepath.Abs(s.cfg.Cert.Dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	caPEM, err := os.ReadFile(filepath.Join(abs, sharedcerts.FileCACert))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", `attachment; filename="ca.crt"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(caPEM)
}

type ipFilterResponse struct {
	ID        string  `json:"id"`
	Status    int16   `json:"status"`
	Field     string  `json:"field"`
	Value     *string `json:"value,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

func toIPFilterResponse(rule models.AccessRule) ipFilterResponse {
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

func (s *Server) handleListIPFilters(w http.ResponseWriter, _ *http.Request) {
	rules, err := s.ruleService.ListRules()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]ipFilterResponse, 0, len(rules))
	for i := range rules {
		items = append(items, toIPFilterResponse(rules[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type ipFilterMutateRequest struct {
	Status int16  `json:"status"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

func (s *Server) buildRuleTarget(field, value string) (services.RuleTarget, error) {
	field = strings.TrimSpace(field)
	switch strings.ToUpper(field) {
	case "ALL", "LOCAL", "REMOTE":
		return s.ruleService.NewCategoryRuleTarget(field)
	case "IP":
		ip, err := sharednetwork.NormalizeIP(value)
		if err != nil {
			return services.RuleTarget{}, err
		}
		return s.ruleService.NewRuleTarget("ip", *ip)
	case "COUNTRY", "REGION", "CITY":
		return s.ruleService.NewRuleTarget(strings.ToLower(field), value)
	default:
		return services.RuleTarget{}, errUnsupportedField(field)
	}
}

type fieldError string

func (e fieldError) Error() string { return string(e) }

func errUnsupportedField(field string) error {
	return fieldError("unsupported field: " + field)
}

func (s *Server) handleUpsertIPFilter(w http.ResponseWriter, r *http.Request) {
	var req ipFilterMutateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != 0 && req.Status != 1 {
		writeError(w, http.StatusBadRequest, "status must be 0 (block) or 1 (allow)")
		return
	}
	target, err := s.buildRuleTarget(req.Field, req.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ruleService.UpsertRule(target, req.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (s *Server) handleDeleteIPFilter(w http.ResponseWriter, r *http.Request) {
	var req ipFilterMutateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != 0 && req.Status != 1 {
		writeError(w, http.StatusBadRequest, "status must be 0 (block) or 1 (allow)")
		return
	}
	target, err := s.buildRuleTarget(req.Field, req.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ruleService.DeleteRule(target, req.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}
