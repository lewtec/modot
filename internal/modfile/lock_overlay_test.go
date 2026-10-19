package modfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplySourceLockOverlayFollowsLockedTag(t *testing.T) {
	t.Parallel()

	sum := loadOverlayLock(t, `{
  "dependencies": [
    {
      "kind": "source",
      "ref": "github:jquery/jquery",
      "depName": "jquery/jquery",
      "packageName": "https://github.com/jquery/jquery",
      "currentValue": "4.0.0",
      "currentDigest": "4f2fae08f23b54ce09322e62e73cce6161b8d3cb",
      "datasource": "git-refs"
    },
    {
      "kind": "source",
      "ref": "github:KhronosGroup/Vulkan-Samples",
      "depName": "KhronosGroup/Vulkan-Samples",
      "packageName": "https://github.com/KhronosGroup/Vulkan-Samples",
      "currentValue": "main",
      "currentDigest": "177edebf0cd7d4f669667e49f052cfb56b17e004",
      "datasource": "git-refs"
    }
  ]
}`)

	jquery := applySourceLockOverlay("jquery", SourceConfig{
		Provider: "github",
		Repo:     "jquery/jquery",
	}, sum)
	require.Equal(t, "4.0.0", jquery.Ref)

	vulkan := applySourceLockOverlay("vulkan_samples", SourceConfig{
		Provider: "github",
		Repo:     "KhronosGroup/Vulkan-Samples",
	}, sum)
	require.Equal(t, "main", vulkan.Ref)

	head := applySourceLockOverlay("capstone_wasm", SourceConfig{
		Provider: "github",
		Repo:     "wasilibs/go-capstone",
		Ref:      "HEAD",
	}, sum)
	require.Equal(t, "HEAD", head.Ref)
	require.Empty(t, head.URL)
}

func TestApplySourceLockOverlayFollowsTagSpec(t *testing.T) {
	t.Parallel()

	const spec = `tag:^v?(?<version>\d+\.\d+\.\d+)$`
	sum := &SumFile{sourceLocks: map[string]LockedSource{
		"jquery": {
			Provider:   "github",
			Repo:       "jquery/jquery",
			Ref:        "4.0.0",
			URL:        "https://codeload.github.com/jquery/jquery/tar.gz/4f2fae08f23b54ce09322e62e73cce6161b8d3cb",
			Versioning: `regex:^v?(?<version>\d+\.\d+\.\d+)$`,
		},
		"vulkan_samples": {
			Provider: "github",
			Repo:     "KhronosGroup/Vulkan-Samples",
			Ref:      "main",
			URL:      "https://codeload.github.com/KhronosGroup/Vulkan-Samples/tar.gz/177edebf0cd7d4f669667e49f052cfb56b17e004",
		},
	}}
	locked, ok := sum.FindSource("jquery")
	require.True(t, ok)
	require.NotEmpty(t, locked.URL)

	jquery := applySourceLockOverlay("jquery", SourceConfig{
		Provider: "github",
		Repo:     "jquery/jquery",
		Ref:      spec,
	}, sum)
	require.Equal(t, "4.0.0", jquery.Ref)
	require.Empty(t, jquery.URL)

	vulkan := applySourceLockOverlay("vulkan_samples", SourceConfig{
		Provider: "github",
		Repo:     "KhronosGroup/Vulkan-Samples",
		Ref:      spec,
	}, sum)
	require.Equal(t, spec, vulkan.Ref)
	require.Empty(t, vulkan.URL)

	invalid := applySourceLockOverlay("jquery", SourceConfig{
		Provider: "github",
		Repo:     "jquery/jquery",
		Ref:      "tag:(",
	}, sum)
	require.Equal(t, "tag:(", invalid.Ref)
	require.Empty(t, invalid.URL)
}

func loadOverlayLock(t *testing.T, body string) *SumFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), "modot.lock.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	sum, err := LoadSumFile(path)
	require.NoError(t, err)
	return sum
}
