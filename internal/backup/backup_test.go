package backup_test

import (
	"os"
	"path/filepath"
	"testing"

	lewtest "github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/modot/internal/backup"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestArchiveAction_RunValidation(t *testing.T) {
	t.Parallel()

	ctx := logging.NewWriterContext(t.Output())

	tests := []struct {
		name    string
		action  backup.ArchiveAction
		wantErr error
	}{
		{
			name:    "missing input and output",
			action:  backup.ArchiveAction{},
			wantErr: backup.ErrArchiveNeedsInputAndOutput,
		},
		{
			name: "unsupported format",
			action: backup.ArchiveAction{
				InputDir: "/tmp/in",
				Output:   "/tmp/out.tar",
				Format:   "zip",
			},
			wantErr: backup.ErrUnsupportedArchiveFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.action.Run(ctx, nil)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestArchiveAction_WritesFinalOnlyOnSuccess(t *testing.T) {
	lewtest.Need(t, "tar")

	ctx := logging.NewWriterContext(t.Output())
	inDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(inDir, "note.txt"), []byte("hello"), 0o644))

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "backup.tar")
	action := backup.ArchiveAction{
		InputDir: inDir,
		Output:   outPath,
		Format:   "tar",
	}
	require.NoError(t, action.Run(ctx, nil), "archive")

	st, err := os.Stat(outPath)
	require.NoError(t, err, "final archive missing")
	require.NotZero(t, st.Size(), "final archive is empty")
	assertNoArchiveTemps(t, outDir)
}

func TestArchiveAction_FailureKeepsExistingOutput(t *testing.T) {
	lewtest.Need(t, "tar")

	ctx := logging.NewWriterContext(t.Output())
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "backup.tar")
	sentinel := []byte("previous-good-archive")
	require.NoError(t, os.WriteFile(outPath, sentinel, 0o644))

	action := backup.ArchiveAction{
		InputDir: filepath.Join(outDir, "does-not-exist"),
		Output:   outPath,
		Format:   "tar",
	}
	require.Error(t, action.Run(ctx, nil), "expected archive of missing input to fail")

	got, err := os.ReadFile(outPath)
	require.NoError(t, err, "existing output should remain")
	require.Equal(t, string(sentinel), string(got))
	assertNoArchiveTemps(t, outDir)
}

func TestRsyncAction_RunValidation(t *testing.T) {
	t.Parallel()

	ctx := logging.NewWriterContext(t.Output())
	err := backup.RsyncAction{}.Run(ctx, nil)
	require.ErrorIs(t, err, backup.ErrRsyncNeedsSrcAndDst)
}

func assertNoArchiveTemps(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.NotContains(t, e.Name(), ".tmp-", "leftover archive temp: %s", e.Name())
	}
}
