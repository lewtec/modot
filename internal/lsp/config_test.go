package lsp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveLanguageExtensionFirst(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Extensions:  map[string]string{".go": "go"},
		LanguageIDs: map[string]string{"python": "python", "go": "go_from_id"},
	}
	normalizeConfig(&cfg)

	require.Equal(t, "go", cfg.ResolveLanguage("file:///x/y/z.go", "python"), "extension should win")
	require.Equal(t, "python", cfg.ResolveLanguage("file:///x/y/z.py", "python"), "language id fallback")
	require.Empty(t, cfg.ResolveLanguage("file:///x/y/z.rs", "rust"), "unmapped")
}

func TestBindingsForOrderAndServerID(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Languages: map[string]map[string]Attachment{
			"go": {
				"99_refactree": {Capabilities: map[string]bool{"references": true}},
				"00_gopls":     {Capabilities: map[string]bool{"hover": true}},
			},
		},
		Servers: map[string]Server{
			"gopls":     {Cmd: []string{"gopls"}},
			"refactree": {Cmd: []string{"refactree"}},
		},
	}
	b := cfg.BindingsFor("go")
	require.Len(t, b, 2)
	require.Equal(t, "gopls", b[0].ServerID)
	require.Equal(t, "00_gopls", b[0].OrderKey)
	require.Equal(t, "refactree", b[1].ServerID)
	require.True(t, b[0].HasCapability("hover"), "gopls caps")
	require.False(t, b[0].HasCapability("references"), "gopls caps")
	// empty caps = all
	all := LanguageBinding{ServerID: "x"}
	require.True(t, all.HasCapability("anything"), "empty caps should allow all")
}

func TestTimeoutDefault(t *testing.T) {
	t.Parallel()
	require.Equal(t, defaultRequestTimeout, (Config{}).Timeout(), "default")
	require.Equal(t, 2*time.Second, (Config{RequestTimeout: "2s"}).Timeout(), "parsed")
	require.Equal(t, defaultRequestTimeout, (Config{RequestTimeout: "nope"}).Timeout(), "bad parse fallback")
}

func TestCapabilityForMethod(t *testing.T) {
	t.Parallel()
	require.Equal(t, "hover", CapabilityForMethod("textDocument/hover"))
	require.Equal(t, "workspaceSymbol", CapabilityForMethod("workspace/symbol"))
}

func TestServerIDFromOrderKey(t *testing.T) {
	t.Parallel()
	require.Equal(t, "gopls", serverIDFromOrderKey("00_gopls"))
	require.Equal(t, "gopls", serverIDFromOrderKey("gopls"))
}
