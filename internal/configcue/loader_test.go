package configcue

import (
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestResolveRuntimeInputs_valid(t *testing.T) {
	cueCtx := cuecontext.New()
	v := cueCtx.CompileString(`{
		inputs: {
			self: {from: "self"}
			localmod: {from: "local:./modules/foo"}
			gh: {from: "github:org/repo", version: "v1.0.0"}
		}
	}`)
	require.NoError(t, v.Err(), "compile cue")

	repoCue := filepath.Join(t.TempDir(), "modot.cue")
	out, err := resolveRuntimeInputs(v, []string{repoCue}, []Layer{{Name: "repo", Path: repoCue}})
	require.NoError(t, err, "resolveRuntimeInputs")
	require.NotEmpty(t, out["self"]["path"], "self path missing: %#v", out["self"])
	got, ok := out["localmod"]["path"].(string)
	require.True(t, ok, "localmod path = %#v", out["localmod"])
	require.True(t, strings.HasSuffix(filepath.Clean(got), filepath.Join("modules", "foo")), "localmod path = %#v", out["localmod"])
	got, ok = out["gh"]["path"].(string)
	require.True(t, ok, "gh path = %#v, want absolute cache path under .cache/modot/sources/github", out["gh"])
	require.True(t, filepath.IsAbs(got), "gh path = %#v, want absolute cache path under .cache/modot/sources/github", out["gh"])
	require.Contains(t, got, filepath.Join(".cache", "modot", "sources", "github"))
}

func TestResolveRuntimeInputs_decodeError(t *testing.T) {
	cueCtx := cuecontext.New()
	// Shape that JSON-decodes but cannot map into inputCfg (string instead of object).
	v := cueCtx.CompileString(`{inputs: {bad: "not-an-object"}}`)
	require.NoError(t, v.Err(), "compile cue")
	_, err := resolveRuntimeInputs(v, nil, nil)
	require.Error(t, err, "expected error for non-object input")
	require.ErrorIs(t, err, ErrDecodeInputs)
}

func TestDecodeReadyMap_skipsIncompleteFileType(t *testing.T) {
	cueCtx := cuecontext.New()
	v := cueCtx.CompileString(`{
		inputs: {self: {from: "self"}}
		file: {
			"x.toml": {
				type: "json" | "toml" | "yaml"
				values: {a: 1}
			}
		}
	}`)
	require.NoError(t, v.Err(), "compile cue")
	_, err := v.MarshalJSON()
	require.Error(t, err, "full marshal should fail on incomplete file type")
	got, err := decodeReadyMap(v)
	require.NoError(t, err, "decodeReadyMap")
	inputs, _ := got["inputs"].(map[string]any)
	self, _ := inputs["self"].(map[string]any)
	require.Equal(t, "self", self["from"], "inputs.self.from = %#v", got["inputs"])
	file, _ := got["file"].(map[string]any)
	entry, _ := file["x.toml"].(map[string]any)
	values, _ := entry["values"].(map[string]any)
	require.True(t, values["a"] == float64(1) || values["a"] == 1, "file values = %#v, want a=1 without requiring type", got["file"])
	require.NotContains(t, entry, "type")
}

func TestResolveRuntimeInputs_empty(t *testing.T) {
	cueCtx := cuecontext.New()
	v := cueCtx.CompileString(`{modules: {}}`)
	require.NoError(t, v.Err(), "compile cue")
	out, err := resolveRuntimeInputs(v, nil, nil)
	require.NoError(t, err, "resolveRuntimeInputs")
	require.Nil(t, out)
}
