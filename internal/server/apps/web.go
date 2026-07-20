package apps

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/controllers"
	"github.com/xiaotiancaipro/nextunnel/internal/server/middleware"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	"go.uber.org/zap"
)

type Web struct {
	Config     *configs.Configs
	Logger     *zap.Logger
	Services   *services.Services
	httpServer *http.Server
	engine     *gin.Engine
}

func (a *Web) Init() error {
	gin.SetMode(gin.ReleaseMode)
	a.engine = gin.New()
	a.engine.Use(gin.Recovery(), middleware.CORS())
	a.initRouters()
	return nil
}

func (a *Web) Start() error {
	addr := fmt.Sprintf("%s:%d", a.Config.ServerWeb.Host, a.Config.ServerWeb.PortOrDefault())
	a.httpServer = &http.Server{
		Addr:              addr,
		Handler:           a.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	a.Logger.Info("web server listening on " + addr)
	if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (a *Web) Stop(ctx context.Context) error {
	if a.httpServer == nil {
		return nil
	}
	return a.httpServer.Shutdown(ctx)
}

func (a *Web) initRouters() {

	front := new(controllers.Front).Init()
	a.engine.NoRoute(front.Index)

	api := a.engine.Group("/api")

	client := controllers.Client{
		ClientService:     a.Services.Client,
		ClientCertService: a.Services.ClientCert,
	}
	api.GET("/clients", client.List)
	api.POST("/clients", client.Create)
	api.DELETE("/clients/:name", client.Delete)

	clientCert := controllers.ClientCert{
		Config:            a.Config,
		ClientService:     a.Services.Client,
		ClientCertService: a.Services.ClientCert,
	}
	api.GET("/clients/:name/sharedcerts", clientCert.List)
	api.GET("/clients/:name/sharedcerts/:id/download", clientCert.Download)
	api.GET("/ca", clientCert.DownloadCA)
	api.POST("/clients/:name/sharedcerts", clientCert.Create)
	api.DELETE("/clients/:name/sharedcerts/:id", clientCert.Delete)

	ipFilter := controllers.IPFilter{
		AccessRuleService: a.Services.AccessRule,
	}
	api.GET("/ip-filters", ipFilter.List)
	api.POST("/ip-filters", ipFilter.Upsert)
	api.DELETE("/ip-filters", ipFilter.Delete)

}
