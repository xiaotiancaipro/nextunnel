package controllers

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var dist embed.FS

type Front struct {
	content    fs.FS
	fileServer http.Handler
}

func (c *Front) Init() *Front {
	content, _ := fs.Sub(dist, "dist")
	return &Front{
		content:    content,
		fileServer: http.FileServer(http.FS(content)),
	}
}

func (c *Front) Index(ctx *gin.Context) {
	path := strings.TrimPrefix(ctx.Request.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	if _, err := fs.Stat(c.content, path); err != nil {
		req := ctx.Request.Clone(ctx.Request.Context())
		req.URL.Path = "/"
		ctx.Request = req
	}
	c.fileServer.ServeHTTP(ctx.Writer, ctx.Request)
}
