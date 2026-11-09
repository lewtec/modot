package view

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/lewtec/lewkit/x/http/asset"
)

// ThemeDefault is the palette when no theme cookie is set.
const ThemeDefault = "light"

// ThemeCookie keeps the chosen daisyUI palette across navigation.
const ThemeCookie = "modot_theme"

// Theme reads the theme cookie, or ThemeDefault.
func Theme(r *http.Request) string {
	if cookie, err := r.Cookie(ThemeCookie); err == nil && (cookie.Value == "light" || cookie.Value == "dark") {
		return cookie.Value
	}
	return ThemeDefault
}

// Handler serves the dashboard. reload is called on every page request so the
// view reflects edits to the config files without restarting the process.
func Handler(reload func(r *http.Request) View) http.Handler {
	load := func(r *http.Request) View {
		v := reload(r)
		v.Theme = Theme(r)
		return v
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, Overview(load(r)))
	})
	mux.HandleFunc("GET /layers", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, Layers(load(r)))
	})
	mux.HandleFunc("GET /modules", func(w http.ResponseWriter, r *http.Request) {
		v := load(r)
		v.ModuleName = r.URL.Query().Get("name")
		if v.ModuleName == "" {
			if mod, ok := selectedModule(v.Report, ""); ok {
				v.ModuleName = mod.Name
			}
		}
		render(w, r, Modules(v))
	})
	mux.HandleFunc("GET /merged", func(w http.ResponseWriter, r *http.Request) {
		v := load(r)
		v.Path = r.URL.Query().Get("path")
		render(w, r, Merged(v))
	})
	mux.HandleFunc("GET /merged.json", func(w http.ResponseWriter, r *http.Request) {
		v := reload(r)
		w.Header().Set("Content-Type", "application/json")
		if v.LoadErr != nil {
			http.Error(w, v.LoadErr.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, r, v.Report.Merged)
	})
	return asset.Mount(mux)
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writeJSON encodes value with two-space indentation. A failure happens after
// the headers are sent, so it can only be logged.
func writeJSON(w http.ResponseWriter, r *http.Request, value map[string]any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		slog.Error("dashboard: encode merged config", "path", r.URL.Path, "error", err)
	}
}
