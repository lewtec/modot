package view_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lewtec/modot/internal/configdash"
	"github.com/lewtec/modot/internal/configdash/view"
	"github.com/stretchr/testify/require"
)

// get is a dashboard request carrying the browser's cookies, if any.
func get(t *testing.T, cookie string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home/", nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	return req
}

// The theme follows the system until someone picks a palette: no cookie means
// "system", which the client turns into light or dark.
func TestThemeDefaultsToSystem(t *testing.T) {
	t.Parallel()
	req := get(t, "")

	require.Equal(t, view.ThemeSystem, view.Theme(req))
	require.Equal(t, view.ThemeDefault, view.Theme(req))
}

func TestThemeReadsCookie(t *testing.T) {
	t.Parallel()
	for _, want := range []string{view.ThemeSystem, view.ThemeLight, view.ThemeDark} {
		req := get(t, view.ThemeCookie+"="+want)

		require.Equal(t, want, view.Theme(req), "cookie %q", want)
	}
}

// An unknown cookie value falls back to following the system rather than
// pinning a palette nobody asked for.
func TestThemeIgnoresUnknownCookie(t *testing.T) {
	t.Parallel()
	req := get(t, view.ThemeCookie+"=solarized")

	require.Equal(t, view.ThemeSystem, view.Theme(req))
}

// The first paint uses a real daisyUI palette; "system" is resolved by the
// client, so the server can only send light.
func TestThemePalette(t *testing.T) {
	t.Parallel()
	require.Equal(t, view.ThemeDark, view.ThemePalette(view.ThemeDark))
	require.Equal(t, view.ThemeLight, view.ThemePalette(view.ThemeLight))
	require.Equal(t, view.ThemeLight, view.ThemePalette(view.ThemeSystem))
}

// Every scope is served, so the switcher and the routes cannot disagree.
func TestScopesCoverEveryMode(t *testing.T) {
	t.Parallel()
	require.Equal(t, []configdash.Scope{
		configdash.ScopeHome, configdash.ScopeCodebase, configdash.ScopeSystem,
	}, view.Scopes())
}

// homeView is the page furniture a request carries before any page-specific
// data is added.
func homeView() view.View {
	var v view.View
	v.Theme = view.ThemeSystem
	v.Scope = configdash.ScopeHome
	v.Report.Scope = configdash.ScopeHome

	return v
}

// The files page serves the tree the loader built, and the sidebar links to it
// from every page, so the route cannot drift from the navigation.
func TestFilesPageServesTree(t *testing.T) {
	t.Parallel()
	loader := view.Loader{
		Reload: func(*http.Request, configdash.Scope) view.View {
			return homeView()
		},
		Files: func(*http.Request, configdash.Scope) configdash.FilesReport {
			var files configdash.FilesReport
			files.Mode = "home"
			files.TargetBase = "~"
			files.Profiles = []string{"home"}
			var shell configdash.MaterialFile
			shell.Target = "~/.bashrc"
			shell.Path = ".bashrc"
			shell.Name = ".bashrc"
			shell.Root = "~"
			shell.Kind = "static"
			shell.Source = "theme"
			shell.Mode = "0644"
			shell.Content = configdash.FileContent{Text: "# hi\n", Size: 5, Lines: 1}
			files.Files = []configdash.MaterialFile{shell}

			return files
		},
	}

	recorder := httptest.NewRecorder()
	view.Handler(configdash.ScopeHome, loader).ServeHTTP(recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home/files/", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, ".bashrc")
	require.Contains(t, body, "theme")
	require.Contains(t, body, `href="/home/files"`)
	// The tree links a file to its own page instead of printing its body in
	// the list.
	require.Contains(t, body, `data-file-link`)
	require.Contains(t, body, `href="/home/file?`)
}

// errFakeModule stands in for a pipeline failure, which carries a module's
// error rather than a typed sentinel.
var errFakeModule = errors.New("plugin module-scanner failed")

// A tree that failed still renders: the page explains what went wrong instead
// of returning a 500.
func TestFilesPageShowsTreeError(t *testing.T) {
	t.Parallel()
	loader := view.Loader{
		Reload: func(*http.Request, configdash.Scope) view.View {
			return homeView()
		},
		Files: func(*http.Request, configdash.Scope) configdash.FilesReport {
			var files configdash.FilesReport
			files.Err = errFakeModule

			return files
		},
	}

	recorder := httptest.NewRecorder()
	view.Handler(configdash.ScopeHome, loader).ServeHTTP(recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home/files/", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "plugin module-scanner failed")
}

// filesView is a loader whose tree holds one shell file, so the content page
// has something real to render.
func filesView() view.Loader {
	return view.Loader{
		Reload: func(*http.Request, configdash.Scope) view.View {
			return homeView()
		},
		Files: func(*http.Request, configdash.Scope) configdash.FilesReport {
			var files configdash.FilesReport
			var shell configdash.MaterialFile
			shell.Target = "~/.bashrc"
			shell.Path = ".bashrc"
			shell.Name = ".bashrc"
			shell.Root = "~"
			shell.Kind = "template"
			shell.Source = "theme"
			shell.Mode = "0644"
			shell.Content = configdash.FileContent{Text: "# hello\nexport PATH=\"$PATH\"\n", Size: 26, Lines: 2}
			files.Files = []configdash.MaterialFile{shell}

			return files
		},
	}
}

// The content page shows one file, highlighted, with the context that says
// where it lands.
func TestFileContentPage(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	view.Handler(configdash.ScopeHome, filesView()).ServeHTTP(recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home/file?path=.bashrc&root=~", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, ".bashrc")
	require.Contains(t, body, "~/.bashrc")
	// Chroma writes classed spans; the palette is a stylesheet the page
	// links, not CSS repeated in the response.
	require.Contains(t, body, `class="chroma"`)
	require.Contains(t, body, `href="/styles/templ.css"`)
	require.NotContains(t, body, ".chroma .k", "the palette is served once, not inlined")
	require.Contains(t, body, "export")
}

// A file the scope does not place is explained, not a 500.
func TestFileContentPageMissing(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	view.Handler(configdash.ScopeHome, filesView()).ServeHTTP(recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home/file?path=nope&root=~", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "no such file in this scope")
}

// The syntax palette is served as one stylesheet, so a page pays for a link
// rather than for a copy of the rules.
func TestSyntaxStylesheet(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	view.Handler(configdash.ScopeHome, filesView()).ServeHTTP(recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/styles/templ.css", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/css")
	css := recorder.Body.String()
	// Every token group names its Chroma class. A group says so when it has
	// no dark twin, which is why the dark check only covers those that want
	// the light color in both themes.
	for _, token := range view.ChromaTokens() {
		require.Contains(t, css, ".chroma ."+token, "token %q has no rule", token)
	}
	// The theme that matters most is the one a reader cannot see without it.
	require.Contains(t, css, `[data-theme="dark"] .chroma .k`)
	require.Contains(t, css, ".chroma{display:block;")
	// The tree's rules are served here too, not inlined into the page that
	// repeats them on every row.
	require.Contains(t, css, "[data-file-tree] .menu li > a")
	require.Contains(t, css, "[data-file-tree] .tree-count")
}

// A page that shows the tree links the stylesheet and carries no CSS of its
// own, so a thousand rows do not pay for a thousand copies of the rules.
func TestFileTreeCarriesNoInlineCSS(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	view.Handler(configdash.ScopeHome, filesView()).ServeHTTP(recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/home/files/", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, "data-file-tree")
	require.Contains(t, body, `href="/styles/templ.css"`)
	require.NotContains(t, body, "<style", "the tree styles are served once")
}
