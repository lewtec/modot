package conda

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// installPrefix is the directory the store keeps after Install returns.
// The store writes into "<final>.<uuidv7>" and renames that onto final,
// so prefix placeholders must name the final directory.
var stagingSuffix = regexp.MustCompile(`\.[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

const defaultPlaceholder = "/opt/anaconda1anaconda2anaconda3"

type prefixFile struct {
	Path        string
	Type        string
	Mode        string
	Placeholder string
}

func installPrefix(destination string) string {
	abs, err := filepath.Abs(destination)
	if err != nil {
		abs = destination
	}
	abs = filepath.Clean(abs)
	if loc := stagingSuffix.FindStringIndex(abs); loc != nil {
		return abs[:loc[0]]
	}
	return abs
}

func extractPackage(archivePath, dest, prefix string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	magic := make([]byte, 4)
	n, err := io.ReadFull(file, magic)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	switch {
	case n >= 2 && magic[0] == 'P' && magic[1] == 'K':
		return extractCondaZip(file, dest, prefix)
	case n >= 3 && magic[0] == 'B' && magic[1] == 'Z' && magic[2] == 'h':
		return extractBzipTar(file, dest, prefix)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownArchive, filepath.Base(archivePath))
	}
}

func extractCondaZip(file *os.File, dest, prefix string) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return err
	}
	var meta []prefixFile
	for _, entry := range reader.File {
		switch {
		case strings.HasPrefix(entry.Name, "pkg-") && strings.HasSuffix(entry.Name, ".tar.zst"):
			if err := withZipEntry(entry, func(r io.Reader) error {
				decoded, err := zstd.NewReader(r)
				if err != nil {
					return err
				}
				defer decoded.Close()
				return extractMembers(decoded, dest)
			}); err != nil {
				return fmt.Errorf("extract %s: %w", entry.Name, err)
			}
		case strings.HasPrefix(entry.Name, "info-") && strings.HasSuffix(entry.Name, ".tar.zst"):
			if err := withZipEntry(entry, func(r io.Reader) error {
				decoded, err := zstd.NewReader(r)
				if err != nil {
					return err
				}
				defer decoded.Close()
				meta, err = readPrefixMeta(decoded)
				return err
			}); err != nil {
				return fmt.Errorf("read %s: %w", entry.Name, err)
			}
		}
	}
	return applyPrefix(dest, prefix, meta)
}

func withZipEntry(entry *zip.File, fn func(io.Reader) error) error {
	reader, err := entry.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return fn(reader)
}

func extractBzipTar(file *os.File, dest, prefix string) error {
	// bzip2.NewReader cannot seek, so buffer the decompressed tar when it
	// carries both payload and info/. Packages are the size of one tool.
	payload, err := io.ReadAll(bzip2.NewReader(file))
	if err != nil {
		return err
	}
	if err := extractMembers(bytes.NewReader(payload), dest); err != nil {
		return err
	}
	meta, err := readPrefixMeta(bytes.NewReader(payload))
	if err != nil {
		return err
	}
	return applyPrefix(dest, prefix, meta)
}

func extractMembers(r io.Reader, dest string) error {
	reader := tar.NewReader(r)
	links := map[string]string{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name, err := safeRel(header.Name)
		if err != nil {
			return err
		}
		if name == "" || name == "info" || strings.HasPrefix(name, "info/") {
			if err := discard(reader); err != nil {
				return err
			}
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, dirMode(header)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := writeMember(target, reader, fileMode(header)); err != nil {
				return err
			}
			links[name] = target
		case tar.TypeSymlink:
			if err := writeSymlink(target, header.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			if err := writeHardlink(dest, target, header.Linkname, links); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: unsupported member %s", ErrUnknownArchive, name)
		}
	}
}

func readPrefixMeta(r io.Reader) ([]prefixFile, error) {
	reader := tar.NewReader(r)
	var pathsJSON []byte
	var hasPrefix []byte
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name, err := safeRel(header.Name)
		if err != nil {
			return nil, err
		}
		switch name {
		case "info/paths.json":
			pathsJSON, err = io.ReadAll(reader)
		case "info/has_prefix":
			hasPrefix, err = io.ReadAll(reader)
		default:
			err = discard(reader)
		}
		if err != nil {
			return nil, err
		}
	}
	if len(pathsJSON) > 0 {
		return prefixFromPaths(pathsJSON)
	}
	return prefixFromHasPrefix(hasPrefix), nil
}

// prefixPlaceholder is either a raw placeholder string (file_mode sits on the
// path entry) or an object with file_mode and placeholder.
type prefixPlaceholder struct {
	FileMode    string
	Placeholder string
}

func (p *prefixPlaceholder) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		return json.Unmarshal(data, &p.Placeholder)
	}
	var obj struct {
		FileMode    string `json:"file_mode"`
		Placeholder string `json:"placeholder"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*p = prefixPlaceholder{FileMode: obj.FileMode, Placeholder: obj.Placeholder}
	return nil
}

func prefixFromPaths(payload []byte) ([]prefixFile, error) {
	var doc struct {
		Paths []struct {
			Path              string            `json:"_path"`
			Alt               string            `json:"path"`
			Type              string            `json:"path_type"`
			FileMode          string            `json:"file_mode"`
			PrefixPlaceholder prefixPlaceholder `json:"prefix_placeholder"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil, fmt.Errorf("decode paths.json: %w", err)
	}
	out := make([]prefixFile, 0, len(doc.Paths))
	for _, item := range doc.Paths {
		if item.PrefixPlaceholder.Placeholder == "" {
			continue
		}
		name := item.Path
		if name == "" {
			name = item.Alt
		}
		mode := item.PrefixPlaceholder.FileMode
		if mode == "" {
			mode = item.FileMode
		}
		if mode == "" {
			mode = "text"
		}
		out = append(out, prefixFile{
			Path:        name,
			Type:        item.Type,
			Mode:        mode,
			Placeholder: item.PrefixPlaceholder.Placeholder,
		})
	}
	return out, nil
}

func prefixFromHasPrefix(payload []byte) []prefixFile {
	var out []prefixFile
	for line := range strings.Lines(string(payload)) {
		fields := strings.Fields(strings.TrimSpace(line))
		switch len(fields) {
		case 1:
			out = append(out, prefixFile{Path: fields[0], Mode: "text", Placeholder: defaultPlaceholder})
		case 3:
			out = append(out, prefixFile{Path: fields[2], Mode: fields[1], Placeholder: fields[0]})
		}
	}
	return out
}

func applyPrefix(dest, prefix string, files []prefixFile) error {
	for _, file := range files {
		if file.Placeholder == "" || file.Path == "" {
			continue
		}
		rel, err := safeRel(file.Path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if file.Type == "softlink" || file.Type == "symlink" {
			if err := rewriteSymlink(target, file, prefix); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(target)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		next, err := replacePlaceholder(data, file.Placeholder, prefix, file.Mode)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		if bytes.Equal(data, next) {
			continue
		}
		mode := os.FileMode(0o644)
		if info, statErr := os.Stat(target); statErr == nil {
			mode = info.Mode().Perm()
		}
		if err := os.WriteFile(target, next, mode); err != nil {
			return err
		}
	}
	return nil
}

func rewriteSymlink(target string, file prefixFile, prefix string) error {
	link, err := os.Readlink(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	next, err := replacePlaceholder([]byte(link), file.Placeholder, prefix, "text")
	if err != nil {
		return err
	}
	if string(next) == link {
		return nil
	}
	if err := os.Remove(target); err != nil {
		return err
	}
	return os.Symlink(string(next), target)
}

var errPrefixLength = errors.New("binary prefix replacement changed file size")

func replacePlaceholder(data []byte, placeholder, prefix, mode string) ([]byte, error) {
	old := []byte(placeholder)
	if !bytes.Contains(data, old) {
		return data, nil
	}
	repl := []byte(prefix)
	if mode != "binary" {
		return bytes.ReplaceAll(data, old, repl), nil
	}
	if len(repl) > len(old) {
		return nil, fmt.Errorf("%w: prefix is %d bytes, placeholder is %d", ErrPrefixTooLong, len(repl), len(old))
	}
	return binaryReplace(data, old, repl)
}

// binaryReplace follows conda's binary prefix rewrite. A placeholder is the
// start of a C string that may continue (linker scripts do) until a NUL.
// Padding goes after that terminator so the tail of the string stays intact.
func binaryReplace(data, old, repl []byte) ([]byte, error) {
	size := len(data)
	out := make([]byte, 0, size)
	for len(data) > 0 {
		at := bytes.Index(data, old)
		if at < 0 {
			out = append(out, data...)
			break
		}
		out = append(out, data[:at]...)
		tail := data[at:]
		end := bytes.IndexByte(tail, 0)
		if end < 0 {
			out = append(out, tail...)
			break
		}
		span := tail[:end+1]
		count := bytes.Count(span, old)
		replaced := bytes.ReplaceAll(span, old, repl)
		pad := (len(old) - len(repl)) * count
		out = append(out, replaced...)
		out = append(out, bytes.Repeat([]byte{0}, pad)...)
		data = tail[end+1:]
	}
	if len(out) != size {
		return nil, errPrefixLength
	}
	return out, nil
}

func safeRel(name string) (string, error) {
	name = path.Clean(strings.TrimPrefix(name, "./"))
	if name == "." || name == "" {
		return "", nil
	}
	if path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
		return "", fmt.Errorf("%w: %s", ErrPathEscapes, name)
	}
	return name, nil
}

func writeMember(target string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, r)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chmod(target, mode)
}

func writeSymlink(target, linkname string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return os.Symlink(linkname, target)
}

func writeHardlink(dest, target, linkname string, links map[string]string) error {
	rel, err := safeRel(linkname)
	if err != nil {
		return err
	}
	source, ok := links[rel]
	if !ok {
		source = filepath.Join(dest, filepath.FromSlash(rel))
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return os.Link(source, target)
}

func fileMode(header *tar.Header) os.FileMode {
	mode := os.FileMode(header.Mode) & os.ModePerm
	if mode == 0 {
		return 0o644
	}
	return mode
}

func dirMode(header *tar.Header) os.FileMode {
	mode := os.FileMode(header.Mode) & os.ModePerm
	if mode == 0 {
		return 0o755
	}
	return mode
}

func discard(r io.Reader) error {
	_, err := io.Copy(io.Discard, r)
	return err
}
