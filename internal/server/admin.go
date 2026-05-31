package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed admin/*
var adminFS embed.FS

func (s *Server) mountAdmin(r chiRouter) {
	sub, err := fs.Sub(adminFS, "admin")
	if err != nil {
		panic("admin ui embed: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
	})
	r.Handle("/admin/*", http.StripPrefix("/admin/", fileServer))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
	})
}

// chiRouter is the minimal route interface used by mountAdmin.
type chiRouter interface {
	Get(pattern string, h http.HandlerFunc)
	Handle(pattern string, h http.Handler)
}
