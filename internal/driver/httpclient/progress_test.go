package httpclient

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestByteProgress(t *testing.T) {
	got := byteProgress(512*1024, 2*1024*1024)
	want := "512.0 KiB / 2.0 MiB"
	require.Equal(t, want, got)
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{-1, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, humanBytes(tt.in), "humanBytes(%d)", tt.in)
	}
}

func TestTaskName(t *testing.T) {
	req := func(raw string) *http.Request {
		u, err := url.Parse(raw)
		require.NoError(t, err)
		return &http.Request{URL: u}
	}

	tests := []struct {
		raw  string
		want string
	}{
		{"https://cdn.example.com/path/bundle.tar.gz", "bundle.tar.gz"},
		{"https://api.github.com/repos/o/r/releases/assets/12345", "github release"},
		{"https://objects.githubusercontent.com/github-production-release-asset-2e65be/abc", "github"},
		{"https://example.com/api/v1/items/99", "example.com"},
		{"https://example.com/", "example.com"},
	}
	for _, tt := range tests {
		got := taskName(req(tt.raw))
		assert.Equal(t, tt.want, got, "taskName(%s)", tt.raw)
	}
}

func TestTaskNameWithLabel(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/o/r/releases/assets/1")
	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	req = req.WithContext(WithTaskLabel(req.Context(), "tool@1.2.3"))
	require.Equal(t, "tool@1.2.3", taskName(req))
}
