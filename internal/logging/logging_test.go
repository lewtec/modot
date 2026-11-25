package logging

import (
	"bytes"
	"log/slog"
	"testing"

	lewlog "github.com/lewtec/lewkit/x/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLogArgs_KeyValuePairs(t *testing.T) {
	got := normalizeLogArgs("stderr", "boom", "context", "lint failed")
	want := []any{"stderr", "boom", "context", "lint failed"}
	assertAnySlice(t, got, want)
}

func TestNormalizeLogArgs_SlogAttr(t *testing.T) {
	got := normalizeLogArgs(slog.String("op", "close"), "path", "/tmp/x")
	want := []any{"op", "close", "path", "/tmp/x"}
	assertAnySlice(t, got, want)
}

func TestNormalizeLogArgs_DanglingKeyDropped(t *testing.T) {
	got := normalizeLogArgs("a", 1, "orphan")
	want := []any{"a", 1}
	assertAnySlice(t, got, want)
}

func TestReportError_KeyValuePairs(t *testing.T) {
	var buf bytes.Buffer
	h := lewlog.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	ctx := NewRootContext(slog.New(h))

	require.True(t, ReportError(ctx, errSentinel{}, "context", "unit test"), "expected ReportError to report non-nil err")
	out := buf.String()
	require.NotEmpty(t, out, "expected log output")
	// Plain handler emits key=value; just sanity-check message and attrs land.
	for _, sub := range []string{"unexpected error", "context", "unit test", "error"} {
		assert.Contains(t, out, sub)
	}
}

func TestReportError_NilErr(t *testing.T) {
	ctx := NewRootContext(slog.Default())
	require.False(t, ReportError(ctx, nil, "context", "should not log"), "expected false for nil err")
}

type errSentinel struct{}

func (errSentinel) Error() string { return "sentinel" }

func assertAnySlice(t *testing.T, got, want []any) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, want[i], got[i], "idx %d", i)
	}
}
