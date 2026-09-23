package sudo

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasew/workspaced/internal/types"
	_ "github.com/lucasew/workspaced/pkg/driver/prelude"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestQueuePathRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		slug string
		want error
	}{
		{slug: "../escape", want: ErrInvalidQueueSlug},
		{slug: "..", want: ErrInvalidQueueSlug},
		{slug: "foo/bar", want: ErrInvalidQueueSlug},
		{slug: "foo\\bar", want: ErrInvalidQueueSlug},
		{slug: "", want: ErrEmptyQueueSlug},
		{slug: ".", want: ErrInvalidQueueSlug},
		{slug: "a/../../etc/passwd", want: ErrInvalidQueueSlug},
	}
	for _, tc := range cases {
		_, err := queuePath(dir, tc.slug)
		require.ErrorIs(t, err, tc.want, "queuePath(%q)", tc.slug)
	}
}

func TestQueuePathAcceptsSimpleSlug(t *testing.T) {
	dir := t.TempDir()
	p, err := queuePath(dir, "abc123")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "abc123.json"), p)
}

func TestEnqueueJailsSlugAndMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := logging.NewWriterContext(t.Output())

	// Malicious slug must not write outside queue dir.
	err := Enqueue(ctx, &types.SudoCommand{
		Slug:    "../escape",
		Command: "true",
		Env:     []string{"SECRET=s3cr3t"},
	})
	require.Error(t, err, "expected error for path-escaping slug")
	// No escape file next to queue parent
	_, err = os.Stat(filepath.Join(home, ".cache/workspaced/escape.json"))
	require.Error(t, err, "escaped write created file outside queue")

	err = Enqueue(ctx, &types.SudoCommand{
		Slug:    "safe1",
		Command: "true",
		Env:     []string{"SECRET=s3cr3t"},
	})
	require.NoError(t, err)
	path := filepath.Join(home, ".cache/workspaced/sudo_queue/safe1.json")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o600), info.Mode().Perm())
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var got types.SudoCommand
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "true", got.Command)
	require.Contains(t, strings.Join(got.Env, "\n"), "SECRET=s3cr3t")

	// Get / Remove use same jail
	_, err = Get("../escape")
	require.Error(t, err, "Get accepted escaping slug")
	require.Error(t, Remove("../escape"), "Remove accepted escaping slug")
	require.NoError(t, Remove("safe1"))
	_, err = os.Stat(path)
	require.ErrorIs(t, err, fs.ErrNotExist, "file still present after Remove")
}

func TestGetQueueDirMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir, err := getQueueDir()
	require.NoError(t, err)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o700), info.Mode().Perm())
}
