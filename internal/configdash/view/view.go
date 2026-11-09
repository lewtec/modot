package view

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/lewtec/lewkit/x/http/asset"
	"github.com/lewtec/modot/internal/configdash"
)

// ThemeSystem means "follow the operating system preference", which only the
// browser can see. It is the default; the page starts light and the client
// script swaps the palette before paint.
const ThemeSystem = "system"

// ThemeLight is the light daisyUI palette.
const ThemeLight = "light"

// ThemeDark is the dark daisyUI palette.
const ThemeDark = "dark"

// ThemePalette turns a theme choice into the palette the page starts with. The
// browser redraws it when the choice is ThemeSystem.
func ThemePalette(theme string) string {
	if theme == ThemeDark {
		return ThemeDark
	}

	return ThemeLight
}

// ThemeDefault is the choice when no theme cookie is set.
const ThemeDefault = ThemeSystem

// ThemeCookie keeps the chosen theme across navigation.
const ThemeCookie = "modot_theme"

// Theme reads the theme cookie, or ThemeDefault. ThemeSystem resolves on the
// client: the server cannot know the machine's preference, and it sends the
// light palette so the page is readable before the script runs.
func Theme(r *http.Request) string {
	cookie, err := r.Cookie(ThemeCookie)
	if err == nil && (cookie.Value == ThemeLight || cookie.Value == ThemeDark || cookie.Value == ThemeSystem) {
		return cookie.Value
	}

	return ThemeDefault
}

// Scopes lists the scopes the dashboard serves, in switcher order.
func Scopes() []configdash.Scope {
	return []configdash.Scope{configdash.ScopeHome, configdash.ScopeCodebase, configdash.ScopeSystem}
}

// page is one dashboard screen.
type page string

const (
	pageOverview page = ""
	pageLayers   page = "layers"
	pageModules  page = "modules"
	pageMerged   page = "merged"
	pageFiles    page = "files"
	pageFile     page = "file"
)

// pageNames lists every page, in sidebar order.
func pageNames() []page {
	return []page{pageOverview, pageLayers, pageModules, pageMerged, pageFiles}
}

// scopeRoute is the URL path for a page in a scope: /home/layers, /codebase.
func scopeRoute(scope configdash.Scope, p page) string {
	path := "/" + string(scope)
	if p != pageOverview {
		path += "/" + string(p)
	}

	return path
}

// Landing picks the scope / redirects to. requested wins when it has layers;
// otherwise the other scope is tried, so `modot dashboard` in a repository
// still shows something useful when the machine also has home layers.
func Landing(requested, other configdash.Scope, homeErr, codebaseErr error) (configdash.Scope, bool) {
	errFor := func(scope configdash.Scope) error {
		if scope == configdash.ScopeHome {
			return homeErr
		}

		return codebaseErr
	}
	if errFor(requested) == nil {
		return requested, true
	}
	if errFor(other) == nil {
		return other, true
	}

	return requested, false
}

// Loader supplies the pages of one scope. Reload is called on every config page
// request and Files only on the materialized-files page, because building the
// file tree renders every template and is much slower than reading the merge.
type Loader struct {
	// Reload returns the view for scope.
	Reload func(r *http.Request, scope configdash.Scope) View
	// Files returns the files scope's apply would place. The content page
	// uses it too, and picks the requested file out of the same tree.
	Files func(r *http.Request, scope configdash.Scope) configdash.FilesReport
}

// Handler serves the dashboard. The loader is called on every page request for
// the scope in the URL, so edits show up without a restart and a scope that
// fails to load still renders its error next to the switcher.
func Handler(requested configdash.Scope, loader Loader) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, scopeRoute(requested, pageOverview), http.StatusFound)
	})
	for _, scope := range Scopes() {
		for _, p := range pageNames() {
			mux.HandleFunc("GET "+scopeRoute(scope, p)+"/{$}", pageHandler(scope, p, loader))
		}
		mergedJSON := func(writer http.ResponseWriter, req *http.Request) {
			board := load(req, scope, loader.Reload)
			writer.Header().Set("Content-Type", "application/json")
			if board.LoadErr != nil {
				http.Error(writer, board.LoadErr.Error(), http.StatusInternalServerError)

				return
			}
			writeJSON(writer, req, board.Report.Merged)
		}
		mux.HandleFunc("GET "+scopeRoute(scope, pageMerged)+".json", mergedJSON)
		mux.HandleFunc("GET "+scopeRoute(scope, pageFile), fileHandler(scope, loader))
	}

	return asset.Mount(mux)
}

// pageHandler serves one page of one scope and fills the selection it needs
// from the query. Only the materialized-files page builds the file tree, so a
// plain navigation never pays for it.
func pageHandler(scope configdash.Scope, p page, loader Loader) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		board := load(req, scope, loader.Reload)
		switch p {
		case pageLayers:
			render(writer, req, Layers(board))
		case pageModules:
			board.ModuleName = req.URL.Query().Get("name")
			if board.ModuleName == "" {
				if mod, ok := selectedModule(board.shown(), ""); ok {
					board.ModuleName = mod.Name
				}
			}
			render(writer, req, Modules(board))
		case pageMerged:
			board.Path = req.URL.Query().Get("path")
			render(writer, req, Merged(board))
		case pageFiles:
			if loader.Files != nil {
				board.Files = loader.Files(req, scope)
			}
			render(writer, req, Files(board))
		case pageOverview:
			render(writer, req, Overview(board))
		}
	}
}

// load renders the view for scope and adds the page furniture.
func load(r *http.Request, scope configdash.Scope, reload func(*http.Request, configdash.Scope) View) View {
	v := reload(r, scope)
	v.Theme = Theme(r)
	v.Scope = scope

	return v
}

// fileHandler serves one file's content. The page is reached from the tree, so
// it rebuilds the tree and picks the file the query names: the tree is the
// only thing that knows what apply would write for a path.
func fileHandler(scope configdash.Scope, loader Loader) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		board := load(req, scope, loader.Reload)
		report := configdash.FilesReport{}
		if loader.Files != nil {
			report = loader.Files(req, scope)
		}
		file, err := configdash.FindFile(report, req.URL.Query().Get("path"), req.URL.Query().Get("root"))
		board.Files = report
		if err != nil {
			board.FileErr = err
		} else {
			board.File = file
		}
		render(writer, req, FileContent(board))
	}
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writeJSON encodes value with two-space indentation. A failure happens after
// the headers are sent, so it can only be logged.
func writeJSON(w http.ResponseWriter, r *http.Request, value map[string]any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err := enc.Encode(value)
	if err != nil {
		slog.Error("dashboard: encode merged config", "path", r.URL.Path, "error", err)
	}
}
