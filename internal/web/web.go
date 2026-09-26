// Package web serves the server-rendered wend.press UI.
//
// No JavaScript build tooling. Templates and static assets are embedded
// into the binary via embed.FS, so `go build` produces a self-contained
// artifact and there is no separate frontend deploy.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Handler struct {
	pages map[string]*template.Template
}

func New() (*Handler, error) {
	base, err := template.ParseFS(templatesFS, "templates/layout.html")
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}

	pages := map[string]*template.Template{}
	for _, name := range []string{"global"} {
		clone, err := base.Clone()
		if err != nil {
			return nil, err
		}
		t, err := clone.ParseFS(templatesFS, "templates/page_"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}
		pages[name] = t
	}

	return &Handler{pages: pages}, nil
}

func (h *Handler) Routes(mux *http.ServeMux) {
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheForever(http.FileServer(http.FS(static)))))

	mux.HandleFunc("GET /{$}", h.render("global"))
}

func (h *Handler) render(page string) http.HandlerFunc {
	t, ok := h.pages[page]
	if !ok {
		panic("unknown page: " + page)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.ExecuteTemplate(w, "layout", map[string]any{
			"Title": "wend.press",
		}); err != nil {
			slog.Error("render", "page", page, "err", err)
		}
	}
}

func cacheForever(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
