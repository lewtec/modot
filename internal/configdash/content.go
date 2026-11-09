package configdash

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lewtec/modot/internal/source"
)

var (
	// ErrBinaryContent means the file is not text, so there is nothing
	// useful to show inline.
	ErrBinaryContent = errors.New("binary file")
	// ErrContentTooLarge means the file is bigger than the viewer reads.
	ErrContentTooLarge = errors.New("file is too large to show")
)

const (
	// headBytes is how much of a file decides whether it is text. It is the
	// prefix a binary signature lives in, and small enough that rejecting an
	// image costs one read.
	headBytes = 4096
	// maxContentBytes is how much of one file the page holds. A rendered
	// config file is kilobytes; anything past this is a data file nobody
	// reads in a browser, and the page says so instead of carrying it.
	maxContentBytes = 256 << 10
)

// FileContent is what one file of the tree contains: the bytes apply would
// write for it, already rendered.
type FileContent struct {
	// Text is the content as text; empty when Err is set.
	Text string
	// Size is the number of bytes apply would write, or -1 when it is not
	// known: a binary file is rejected from its head, so counting the rest
	// would cost the work the cap exists to avoid. When Truncated is set the
	// read stopped early and this is only how much was read.
	Size int
	// Lines is how many lines Text has.
	Lines int
	// Truncated is set when the file is bigger than the viewer reads and
	// Text holds only its beginning.
	Truncated bool
	// Err says why there is no text: a binary file, or one too large.
	Err error
}

// HasText reports whether there is content to show.
func (c FileContent) HasText() bool { return c.Text != "" }

// readContent reads one file. It returns the bytes apply would write, so a
// template shows its rendered form; the tree already rendered it, so this is a
// copy, not a render.
func readContent(file source.File) (FileContent, error) {
	reader, err := file.Reader()
	if err != nil {
		return FileContent{}, fmt.Errorf("read %s: %w", file.RelPath(), err)
	}
	defer reader.Close() //nolint:errcheck // a failed close on a read has nothing to add

	// The head is what decides whether the file is text at all, so it is read
	// before anything is capped: an image's size is worth saying even though
	// nothing of it is shown.
	head, err := io.ReadAll(io.LimitReader(reader, headBytes))
	if err != nil {
		return FileContent{}, fmt.Errorf("read %s: %w", file.RelPath(), err)
	}
	if isBinary(head) {
		return FileContent{Text: "", Size: -1, Lines: 0, Truncated: false, Err: ErrBinaryContent}, nil
	}
	data := head
	if len(head) == headBytes {
		rest, restErr := io.ReadAll(io.LimitReader(reader, maxContentBytes+1-headBytes))
		if restErr != nil {
			return FileContent{}, fmt.Errorf("read %s: %w", file.RelPath(), restErr)
		}
		data = append(head, rest...)
	}
	content := FileContent{Text: "", Size: len(data), Lines: 0, Truncated: false, Err: nil}
	if len(data) > maxContentBytes {
		// The file is bigger than the viewer reads, so this is how far the
		// read got, not the file's size. Reading the rest only to count it
		// would cost the work the cap exists to avoid.
		content.Truncated = true
		data = data[:maxContentBytes]
	}
	content.Text = string(data)
	content.Lines = bytes.Count(data, []byte("\n"))
	if content.Lines == 0 || !bytes.HasSuffix(data, []byte("\n")) {
		content.Lines++
	}

	return content, nil
}

// isBinary reports whether data is not text. A NUL byte is the usual signal;
// so is a prefix that is not valid UTF-8, which catches a JPEG or a PNG header
// without scanning the whole image.
func isBinary(data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	if len(data) == 0 {
		return false
	}
	head := data
	if len(head) > 4096 {
		head = head[:4096]
	}
	// A multibyte rune may straddle the cut, so drop a partial tail.
	for len(head) > 0 && !utf8.Valid(head) {
		head = head[:len(head)-1]
	}

	return len(head) == 0
}

// contentErr explains a missing body in the reader's terms.
func contentErr(content FileContent) string {
	switch {
	case errors.Is(content.Err, ErrBinaryContent):
		return "binary file, not shown as text"
	case content.Err != nil:
		return content.Err.Error()
	case content.Truncated:
		return fmt.Sprintf("showing the first %s of %s", byteSize(maxContentBytes), byteSize(content.Size))
	}

	return ""
}

// HumanSize formats a byte count the way a reader skims it. A negative size
// is one the reader did not measure, and says so instead of printing a number
// that would be wrong.
func HumanSize(size int) string {
	if size < 0 {
		return "size unknown"
	}

	return byteSize(size)
}

// byteSize formats a size the way a reader skims it.
func byteSize(size int) string {
	switch {
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/(1<<10))
	}

	return fmt.Sprintf("%d B", size)
}

// contentLang is a rough language for the code block, so a reader gets the
// right shape even though there is no highlighter here.
func contentLang(path string) string {
	switch {
	case strings.HasSuffix(path, ".json"):
		return "json"
	case strings.HasSuffix(path, ".toml"):
		return "toml"
	case strings.HasSuffix(path, ".yml"), strings.HasSuffix(path, ".yaml"):
		return "yaml"
	case strings.HasSuffix(path, ".cue"):
		return "cue"
	case strings.HasSuffix(path, ".sh"):
		return "shell"
	case strings.HasSuffix(path, ".md"):
		return "markdown"
	}

	return "text"
}
