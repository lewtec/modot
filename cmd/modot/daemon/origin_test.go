package daemon

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckOriginAllowsLocalClients(t *testing.T) {
	for _, origin := range []string{"", "null"} {
		r := &http.Request{Header: http.Header{}}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		require.True(t, upgrader.CheckOrigin(r), "CheckOrigin(%q) = false, want true", origin)
	}
}

func TestCheckOriginRejectsForeignBrowserOrigin(t *testing.T) {
	r := &http.Request{Header: http.Header{"Origin": []string{"https://evil.example"}}}
	require.False(t, upgrader.CheckOrigin(r), "CheckOrigin(evil) = true, want false")
}
