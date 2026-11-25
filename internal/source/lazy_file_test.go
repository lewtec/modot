package source

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trackingCloser records whether Close was called.
type trackingCloser struct {
	io.ReadCloser
	closed *bool
}

func (t *trackingCloser) Close() error {
	*t.closed = true
	return t.ReadCloser.Close()
}

func TestConcatenatedFileReaderCloseClosesComponents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, "b.txt"),
	}
	for i, p := range paths {
		content := []byte{'A' + byte(i)}
		require.NoError(t, os.WriteFile(p, content, 0o644), "write %s", p)
	}

	closed := make([]bool, len(paths))
	components := make([]File, len(paths))
	for i, p := range paths {
		sf := &StaticFile{
			BasicFile: BasicFile{
				RelPathStr:    filepath.Base(p),
				TargetBaseDir: dir,
				FileMode:      0o644,
				FileType:      TypeStatic,
			},
			AbsPath: p,
		}
		// Wrap via a custom File that tracks Close through Reader.
		components[i] = &trackingFile{StaticFile: sf, closed: &closed[i]}
	}

	cf := &ConcatenatedFile{
		BasicFile: BasicFile{
			RelPathStr:    "out",
			TargetBaseDir: dir,
			FileMode:      0o644,
			FileType:      TypeDotD,
		},
		Components: components,
	}

	r, err := cf.Reader()
	require.NoError(t, err, "Reader")
	got, err := io.ReadAll(r)
	require.NoError(t, err, "ReadAll")
	require.Equal(t, "A\nB", string(got))
	// Partial read path already finished; Close must still close components.
	require.NoError(t, r.Close(), "Close")
	for i, c := range closed {
		assert.True(t, c, "component %d not closed", i)
	}
}

func TestConcatenatedFileCloseWithoutFullRead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "big.txt")
	require.NoError(t, os.WriteFile(p, []byte("hello world"), 0o644), "write")
	var closed bool
	cf := &ConcatenatedFile{
		BasicFile: BasicFile{RelPathStr: "out", TargetBaseDir: dir, FileMode: 0o644, FileType: TypeDotD},
		Components: []File{
			&trackingFile{
				StaticFile: &StaticFile{
					BasicFile: BasicFile{RelPathStr: "big.txt", TargetBaseDir: dir, FileMode: 0o644, FileType: TypeStatic},
					AbsPath:   p,
				},
				closed: &closed,
			},
		},
	}
	r, err := cf.Reader()
	require.NoError(t, err, "Reader")
	buf := make([]byte, 1)
	_, err = r.Read(buf)
	require.NoError(t, err, "Read")
	require.NoError(t, r.Close(), "Close")
	require.True(t, closed, "component not closed after partial read + Close")
}

// trackingFile is a StaticFile whose Reader returns a closer that records Close.
type trackingFile struct {
	*StaticFile
	closed *bool
}

func (f *trackingFile) Reader() (io.ReadCloser, error) {
	r, err := f.StaticFile.Reader()
	if err != nil {
		return nil, err
	}
	return &trackingCloser{ReadCloser: r, closed: f.closed}, nil
}
