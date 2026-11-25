package deployer

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/source"
	"github.com/stretchr/testify/require"
)

type errReadCloser struct {
	err error
}

func (e errReadCloser) Read([]byte) (int, error) { return 0, e.err }
func (e errReadCloser) Close() error             { return nil }

type failingReaderFile struct {
	source.BasicFile
	openErr error
	readErr error
}

func (f *failingReaderFile) Reader() (io.ReadCloser, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return errReadCloser{err: f.readErr}, nil
}

func TestExecuteKeepsExistingOnCopyError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "managed.txt")
	// atomicfile.Write leaves the prior regular file in place when the new
	// write fails before commit.
	require.NoError(t, os.WriteFile(target, []byte("good"), 0o644))

	actions := []Action{{
		Type:   ActionUpdate,
		Target: target,
		Desired: DesiredState{
			File: &failingReaderFile{
				BasicFile: source.BasicFile{
					RelPathStr:    "managed.txt",
					TargetBaseDir: dir,
					FileMode:      0o644,
					Info:          "test:failing-reader",
					FileType:      source.TypeStatic,
				},
				readErr: errors.New("read boom"),
			},
		},
	}}
	state := &State{Files: map[string]ManagedInfo{
		target: {SourceInfo: "test:old"},
	}}

	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	_ = g
	err := NewExecutor().Execute(ctx, actions, state)
	require.Error(t, err, "expected copy error")
	got, statErr := os.ReadFile(target)
	require.NoError(t, statErr, "expected prior file kept (execute=%v)", err)
	require.Equal(t, "good", string(got), "prior content lost")
	require.Contains(t, state.Files, target, "state should not drop managed entry when apply fails")
}

func TestExecuteRemovesEmptyFileOnReaderError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "managed.txt")

	actions := []Action{{
		Type:   ActionCreate,
		Target: target,
		Desired: DesiredState{
			File: &failingReaderFile{
				BasicFile: source.BasicFile{
					RelPathStr:    "managed.txt",
					TargetBaseDir: dir,
					FileMode:      0o644,
					Info:          "test:open-fail",
					FileType:      source.TypeStatic,
				},
				openErr: errors.New("open boom"),
			},
		},
	}}
	state := &State{Files: map[string]ManagedInfo{}}

	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	_ = g
	err := NewExecutor().Execute(ctx, actions, state)
	require.Error(t, err, "expected reader error")
	_, statErr := os.Stat(target)
	require.ErrorIs(t, statErr, os.ErrNotExist, "empty file still present after reader error (execute=%v)", err)
}

func TestExecuteWritesRegularFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "managed.txt")
	content := []byte("hello apply\n")

	actions := []Action{{
		Type:   ActionCreate,
		Target: target,
		Desired: DesiredState{
			File: &source.BufferFile{
				BasicFile: source.BasicFile{
					RelPathStr:    "managed.txt",
					TargetBaseDir: dir,
					FileMode:      0o644,
					Info:          "test:buffer",
					FileType:      source.TypeStatic,
				},
				Content: content,
			},
		},
	}}
	state := &State{Files: map[string]ManagedInfo{}}

	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	_ = g
	require.NoError(t, NewExecutor().Execute(ctx, actions, state))
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, content, got)
	require.Contains(t, state.Files, target)
	require.Equal(t, "test:buffer", state.Files[target].SourceInfo)
}

func TestExecuteIgnoredCreateOmitsState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "placed.md")
	content := []byte("hello ignore\n")
	ex := NewExecutor()
	ex.Ignore = func(path string) bool { return path == target }

	actions := []Action{{
		Type:   ActionCreate,
		Target: target,
		Desired: DesiredState{
			File: &source.BufferFile{
				BasicFile: source.BasicFile{
					RelPathStr:    "placed.md",
					TargetBaseDir: dir,
					FileMode:      0o644,
					Info:          "module:place (placed.md)",
					FileType:      source.TypeStatic,
				},
				Content: content,
			},
		},
	}}
	state := &State{Files: map[string]ManagedInfo{
		target: {SourceInfo: "old"},
	}}

	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	_ = g
	require.NoError(t, ex.Execute(ctx, actions, state))
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, content, got)
	require.NotContains(t, state.Files, target, "ignored path still in state")
}
