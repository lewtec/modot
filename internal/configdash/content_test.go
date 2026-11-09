package configdash

import (
	"strings"
	"testing"

	"github.com/lewtec/modot/internal/source"
	"github.com/stretchr/testify/require"
)

// bufferFile is a file with in-memory content, the way the tree holds a
// rendered template.
func bufferFile(rel string, data []byte) source.File {
	file := &source.BufferFile{Content: data}
	file.RelPathStr = rel

	return file
}

// A text file is shown as apply would write it, and its line count is the
// number of lines a reader sees.
func TestReadContentText(t *testing.T) {
	t.Parallel()
	content, err := readContent(bufferFile("file.txt", []byte("one\ntwo\n")))
	require.NoError(t, err, "readContent")
	require.Equal(t, "one\ntwo\n", content.Text)
	require.Equal(t, 2, content.Lines)
	require.Equal(t, 8, content.Size)
	require.False(t, content.Truncated)
	require.NoError(t, content.Err)
	require.Empty(t, contentErr(content))
}

// A file without a trailing newline still counts the line it ends with.
func TestReadContentNoTrailingNewline(t *testing.T) {
	t.Parallel()
	content, err := readContent(bufferFile("file.txt", []byte("one\ntwo")))
	require.NoError(t, err, "readContent")
	require.Equal(t, 2, content.Lines)
}

// An image is not text: the page says so instead of showing bytes as a
// string of replacement characters.
func TestReadContentBinary(t *testing.T) {
	t.Parallel()
	content, err := readContent(bufferFile("image.png", []byte("\x89PNG\r\n\x1a\n\x00\x00")))
	require.NoError(t, err, "readContent")
	require.ErrorIs(t, content.Err, ErrBinaryContent)
	require.Empty(t, content.Text)
	require.Contains(t, contentErr(content), "binary file")
}

// A file bigger than the viewer reads shows its beginning and says it was
// cut, so a reader knows the page is not the whole file.
func TestReadContentTruncated(t *testing.T) {
	t.Parallel()
	body := strings.Repeat("a", maxContentBytes+1024)

	content, err := readContent(bufferFile("big.txt", []byte(body)))
	require.NoError(t, err, "readContent")
	require.True(t, content.Truncated)
	require.Len(t, content.Text, maxContentBytes)
	require.Equal(t, maxContentBytes+1, content.Size, "size is how far the capped read got")
	require.Contains(t, contentErr(content), "first")
}

// A file that cannot be opened is an error naming the file, so the page can
// say which one it lost instead of failing the whole tree.
func TestReadContentErrorNamesFile(t *testing.T) {
	t.Parallel()
	file := &source.StaticFile{}
	file.RelPathStr = "nothing.txt"
	file.AbsPath = "/nonexistent/nothing.txt"

	_, err := readContent(file)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nothing.txt")
}

// Sizes are formatted the way a reader skims them.
func TestHumanSize(t *testing.T) {
	t.Parallel()
	require.Equal(t, "512 B", HumanSize(512))
	require.Equal(t, "1.5 KB", HumanSize(1536))
	require.Equal(t, "2.0 MB", HumanSize(2<<20))
}

// A small text file is not binary, including one that is all whitespace.
func TestIsBinaryEmptyAndWhitespace(t *testing.T) {
	t.Parallel()
	require.False(t, isBinary(nil))
	require.False(t, isBinary([]byte("  \n\t\n")))
	require.True(t, isBinary([]byte{0x00}))
}
