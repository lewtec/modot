package view_test

import (
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
