package version

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatform(t *testing.T) {
	got := Platform()
	wantPrefix := runtime.GOOS + "-" + runtime.GOARCH
	require.True(t, got == wantPrefix || strings.HasPrefix(got, wantPrefix+"-"), "Platform() = %q, want %q or %q-<microarch>", got, wantPrefix, wantPrefix)
	// No spaces: separate token for --version.
	require.NotContains(t, got, " ", "Platform() must be a single token, got %q", got)
}

func TestVersionStringHasSpaceSeparatedPlatform(t *testing.T) {
	got := VersionString()
	parts := strings.Split(got, " ")
	require.Len(t, parts, 2, "VersionString() = %q, want exactly two space-separated tokens", got)
	require.Equal(t, GetBuildID(), parts[0])
	require.Equal(t, Platform(), parts[1])
}

func TestGetBuildIDNoSpace(t *testing.T) {
	require.NotContains(t, GetBuildID(), " ", "GetBuildID() must stay filename-safe, got %q", GetBuildID())
}
