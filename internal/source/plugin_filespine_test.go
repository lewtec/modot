package source

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/pkg/filespine"
	"github.com/lucasew/workspaced/pkg/logging"
)

func TestFileSpineLowersDotD(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	p := NewFileSpinePlugin(&configcue.Config{}, home)
	in := []File{
		&BufferFile{
			BasicFile: BasicFile{RelPathStr: ".bashrc.d.tmpl/20-b.sh", TargetBaseDir: home, FileMode: 0o644, FileType: TypeStatic},
			Content:   []byte("b"),
		},
		&BufferFile{
			BasicFile: BasicFile{RelPathStr: ".bashrc.d.tmpl/10-a.sh", TargetBaseDir: home, FileMode: 0o644, FileType: TypeStatic},
			Content:   []byte("a"),
		},
	}
	out, err := p.Process(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d want 1", len(out))
	}
	if out[0].RelPath() != ".bashrc" {
		t.Fatalf("rel=%q", out[0].RelPath())
	}
	r, err := out[0].Reader()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a\nb" {
		t.Fatalf("content=%q", got)
	}
}

func TestFileSpineMergesCueLines(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	cuePath := filepath.Join(t.TempDir(), "workspaced.cue")
	src := `package workspaced
workspaced: {
	file: home: ".bashrc": {
		type: "lines"
		values: {"00-cue": "from-cue"}
	}
}
`
	if err := os.WriteFile(cuePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := configcue.LoadFiles(ctx, []string{cuePath})
	if err != nil {
		t.Fatal(err)
	}
	p := NewFileSpinePlugin(cfg, home)
	in := []File{
		&BufferFile{
			BasicFile: BasicFile{RelPathStr: ".bashrc.d.tmpl/10-mod.sh", TargetBaseDir: home, FileMode: 0o644},
			Content:   []byte("from-mod"),
		},
	}
	out, err := p.Process(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	r, err := out[0].Reader()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "from-cue\nfrom-mod" {
		t.Fatalf("content=%q", got)
	}
}

func TestFileSpineTypeConflict(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	dir := t.TempDir()
	cuePath := filepath.Join(dir, "workspaced.cue")
	if err := os.WriteFile(cuePath, []byte(`package workspaced
workspaced: file: home: "x": {type: "lines", values: {a: "1"}}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := configcue.LoadFiles(ctx, []string{cuePath})
	if err != nil {
		t.Fatal(err)
	}
	p := NewFileSpinePlugin(cfg, home)
	_, err = p.Process(ctx, []File{
		&BufferFile{
			BasicFile: BasicFile{RelPathStr: "x", TargetBaseDir: home, FileMode: 0o644},
			Content:   []byte("text"),
		},
	})
	if err == nil {
		t.Fatal("expected type conflict")
	}
}

func TestFileSpineStaticRef(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	src := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(src, []byte("[user]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewFileSpinePlugin(&configcue.Config{}, home)
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: ".gitconfig", TargetBaseDir: home, FileMode: 0o644, FileType: TypeStatic},
			AbsPath:   src,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	sf, ok := out[0].(*StaticFile)
	if !ok {
		t.Fatalf("type %T", out[0])
	}
	if sf.AbsPath != src {
		t.Fatalf("abs=%q", sf.AbsPath)
	}
	r, err := sf.Reader()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "[user]\n" {
		t.Fatalf("content=%q", got)
	}
}

func TestFileSpineKeepsSymlink(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	p := NewFileSpinePlugin(&configcue.Config{}, home)
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: ".link", TargetBaseDir: home, FileType: TypeStatic},
			AbsPath:   link,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Type() != TypeSymlink {
		t.Fatalf("type=%s", out[0].Type())
	}
	got, err := out[0].LinkTarget()
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("link=%q want %q", got, target)
	}
}

func TestFileSpineEtcUsesFixedBase(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	src := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(src, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewFileSpinePlugin(systemConfig(t), home)
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: "hosts", TargetBaseDir: "/etc", FileType: TypeStatic},
			AbsPath:   src,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].TargetBase() != "/etc" || out[0].RelPath() != "hosts" {
		t.Fatalf("target=%s rel=%s", out[0].TargetBase(), out[0].RelPath())
	}
}

func TestFileSpineSameRelPathOnTwoProfiles(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	homeSrc := filepath.Join(t.TempDir(), "hosts-home")
	etcSrc := filepath.Join(t.TempDir(), "hosts-etc")
	if err := os.WriteFile(homeSrc, []byte("home\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(etcSrc, []byte("etc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewFileSpinePlugin(systemConfig(t), home)
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: "hosts", TargetBaseDir: home, FileType: TypeStatic},
			AbsPath:   homeSrc,
		},
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: "hosts", TargetBaseDir: "/etc", FileType: TypeStatic},
			AbsPath:   etcSrc,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	got := map[string]string{}
	for _, f := range out {
		r, err := f.Reader()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		got[f.TargetBase()] = string(body)
	}
	if got[home] != "home\n" || got["/etc"] != "etc\n" {
		t.Fatalf("bodies=%v", got)
	}
}

func systemConfig(t *testing.T) *configcue.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "workspaced.cue")
	if err := os.WriteFile(path, []byte("package workspaced\nworkspaced: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := configcue.LoadFilesMode(logging.NewWriterContext(t.Output()), []string{path}, filespine.ModeSystem)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestFileSpineRejectsNestedDotD(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	p := NewFileSpinePlugin(&configcue.Config{}, home)
	_, err := p.Process(ctx, []File{
		&BufferFile{
			BasicFile: BasicFile{RelPathStr: ".bashrc.d.tmpl/sub/10.sh", TargetBaseDir: home},
			Content:   []byte("a"),
		},
	})
	if err == nil {
		t.Fatal("expected path error")
	}
}
