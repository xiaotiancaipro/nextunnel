package controllers

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist/*
var dist embed.FS

func Handler() (http.Handler, error) {
	content, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, err
	}
	return spaFileServer(content), nil
}

func spaFileServer(content fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(content))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(content, path); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
