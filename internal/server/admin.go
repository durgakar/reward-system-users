package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed admin/*
var adminFS embed.FS

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(adminFS, "admin")
	if err != nil {
		http.Error(w, "admin ui unavailable", http.StatusInternalServerError)
		return
	}
	path := r.URL.Path
	if path == "/admin" || path == "/admin/" {
		path = "/admin/index.html"
	} else {
		path = path[len("/admin"):]
		if path == "" {
			path = "/index.html"
		}
	}
	data, err := fs.ReadFile(sub, path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if len(path) >= 5 && path[len(path)-5:] == ".html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else if len(path) >= 3 && path[len(path)-3:] == ".js" {
		w.Header().Set("Content-Type", "application/javascript")
	} else if len(path) >= 4 && path[len(path)-4:] == ".css" {
		w.Header().Set("Content-Type", "text/css")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
