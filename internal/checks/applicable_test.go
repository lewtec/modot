package checks_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCheck struct {
	name string
	err  error
}

func (s stubCheck) Name() string                         { return s.name }
func (s stubCheck) Detect(context.Context, string) error { return s.err }

func TestApplicable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644))

	detectErr := errors.New("boom")
	items := []checks.Check{
		stubCheck{name: "ok", err: nil},
		stubCheck{name: "skip", err: checks.ErrNotApplicable},
		stubCheck{name: "fail", err: detectErr},
	}

	var skips []string
	got := checks.Applicable(t.Context(), dir, items, func(name, reason string, err error) {
		skips = append(skips, name+":"+reason)
		if name == "fail" {
			assert.ErrorIs(t, err, detectErr)
		}
	})

	require.Equal(t, []string{"ok"}, names(got))
	require.Equal(t, []string{"skip:not applicable", "fail:detect failed"}, skips)
}

func names(items []checks.Check) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Name()
	}
	return out
}
