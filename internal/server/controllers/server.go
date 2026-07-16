package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Server struct {
	version           string
	cfg               *configs.Configs
	logger            *zap.Logger
	clientService     *services.ClientRegistry
	clientCertService *services.ClientCertRegistry
	ruleService       *services.AccessRule
	httpServer        *http.Server
}

func NewServer(version string, cfg *configs.Configs, db *gorm.DB, logger *zap.Logger) *Server {
	return &Server{
		version:           version,
		cfg:               cfg,
		logger:            logger,
		clientService:     services.NewClientRegistry(db),
		clientCertService: services.NewClientCertRegistry(db, cfg.Cert.Dir, cfg.Cert.Host),
		ruleService:       services.NewAccessRule(db),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	uiHandler, err := Handler()
	if err != nil {
		return fmt.Errorf("initialize web ui: %w", err)
	}
	mux.Handle("/", uiHandler)

	addr := fmt.Sprintf("0.0.0.0:%d", s.cfg.Web.PortOrDefault())
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	s.logger.Info("Web management listening on " + addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
