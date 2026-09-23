package source

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, ".bashrc", out[0].RelPath())
	r, err := out[0].Reader()
	require.NoError(t, err)
	defer r.Close()
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "a\nb", string(got))
}

func TestFileSpineMergesCueLines(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	cuePath := filepath.Join(t.TempDir(), "workspaced.cue")
	src := `package workspaced
file: home: ".bashrc": {
	type: "lines"
	values: {"00-cue": "from-cue"}
}
`
	require.NoError(t, os.WriteFile(cuePath, []byte(src), 0o644))
	cfg, err := configcue.LoadFiles(ctx, []string{cuePath})
	require.NoError(t, err)
	p := NewFileSpinePlugin(cfg, home)
	in := []File{
		&BufferFile{
			BasicFile: BasicFile{RelPathStr: ".bashrc.d.tmpl/10-mod.sh", TargetBaseDir: home, FileMode: 0o644},
			Content:   []byte("from-mod"),
		},
	}
	out, err := p.Process(ctx, in)
	require.NoError(t, err)
	require.Len(t, out, 1)
	r, err := out[0].Reader()
	require.NoError(t, err)
	defer r.Close()
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "from-cue\nfrom-mod", string(got))
}

func TestFileSpineTypeConflict(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	dir := t.TempDir()
	cuePath := filepath.Join(dir, "workspaced.cue")
	require.NoError(t, os.WriteFile(cuePath, []byte(`package workspaced
file: home: "x": {type: "lines", values: {a: "1"}}
`), 0o644))
	cfg, err := configcue.LoadFiles(ctx, []string{cuePath})
	require.NoError(t, err)
	p := NewFileSpinePlugin(cfg, home)
	_, err = p.Process(ctx, []File{
		&BufferFile{
			BasicFile: BasicFile{RelPathStr: "x", TargetBaseDir: home, FileMode: 0o644},
			Content:   []byte("text"),
		},
	})
	require.Error(t, err, "expected type conflict")
}

func TestFileSpineStaticRef(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	src := filepath.Join(t.TempDir(), "gitconfig")
	require.NoError(t, os.WriteFile(src, []byte("[user]\n"), 0o644))
	p := NewFileSpinePlugin(&configcue.Config{}, home)
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: ".gitconfig", TargetBaseDir: home, FileMode: 0o644, FileType: TypeStatic},
			AbsPath:   src,
		},
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	sf, ok := out[0].(*StaticFile)
	require.True(t, ok, "type %T", out[0])
	require.Equal(t, src, sf.AbsPath)
	r, err := sf.Reader()
	require.NoError(t, err)
	defer r.Close()
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "[user]\n", string(got))
}

func TestPlainFileKeepsBundleInfo(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	src := filepath.Join(t.TempDir(), "icon.svg")
	require.NoError(t, os.WriteFile(src, []byte("<svg/>"), 0o644))
	out, err := composeApply(ctx, destRequest{
		targetBase: home,
		files: []File{
			&StaticFile{
				BasicFile: BasicFile{
					RelPathStr:    "icon.svg",
					TargetBaseDir: home,
					Info:          "module:icons bundle:abc (icon.svg)",
					FileType:      TypeStatic,
				},
				AbsPath: src,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, out.Files(), 1)
	require.Equal(t, "module:icons bundle:abc (icon.svg)", out.Files()[0].SourceInfo())
}

func TestComposeApplyStopsWhenCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(logging.NewWriterContext(t.Output()))
	cancel()
	home := t.TempDir()
	_, err := composeApply(ctx, destRequest{
		targetBase: home,
		files: []File{
			&BufferFile{
				BasicFile: BasicFile{RelPathStr: "plain.txt", TargetBaseDir: home},
				Content:   []byte("x"),
			},
		},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestFileSpineNestedTargetStaysInHome(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	p := NewFileSpinePlugin(&configcue.Config{}, home)
	out, err := p.Process(ctx, []File{
		&BufferFile{
			BasicFile: BasicFile{
				RelPathStr:    "dconf.marker",
				TargetBaseDir: filepath.Join(home, ".config", "workspaced"),
				FileMode:      0o644,
			},
			Content: []byte("abc"),
		},
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, home, out[0].TargetBase())
	require.Equal(t, ".config/workspaced/dconf.marker", out[0].RelPath())
}

func TestFileSpineKeepsSymlink(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(target, link))
	p := NewFileSpinePlugin(&configcue.Config{}, home)
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: ".link", TargetBaseDir: home, FileType: TypeStatic},
			AbsPath:   link,
		},
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, TypeSymlink, out[0].Type())
	got, err := out[0].LinkTarget()
	require.NoError(t, err)
	require.Equal(t, target, got)
}

func TestFileSpineEtcUsesFixedBase(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	src := filepath.Join(t.TempDir(), "hosts")
	require.NoError(t, os.WriteFile(src, []byte("127.0.0.1 localhost\n"), 0o644))
	p := NewFileSpinePlugin(systemConfig(t), "/")
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: "hosts", TargetBaseDir: "/etc", FileType: TypeStatic},
			AbsPath:   src,
		},
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "/", out[0].TargetBase())
	require.Equal(t, "etc/hosts", out[0].RelPath())
}

func TestFileSpineSameRelPathOnTwoProfiles(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	prefix := t.TempDir()
	etcBase := filespine.ApplyDir("etc", prefix)
	homeSrc := filepath.Join(t.TempDir(), "hosts-home")
	etcSrc := filepath.Join(t.TempDir(), "hosts-etc")
	require.NoError(t, os.WriteFile(homeSrc, []byte("home\n"), 0o644))
	require.NoError(t, os.WriteFile(etcSrc, []byte("etc\n"), 0o644))
	p := NewFileSpinePlugin(systemConfig(t), prefix)
	out, err := p.Process(ctx, []File{
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: "hosts", TargetBaseDir: prefix, FileType: TypeStatic},
			AbsPath:   homeSrc,
		},
		&StaticFile{
			BasicFile: BasicFile{RelPathStr: "hosts", TargetBaseDir: etcBase, FileType: TypeStatic},
			AbsPath:   etcSrc,
		},
	})
	require.NoError(t, err)
	require.Len(t, out, 2)
	got := map[string]string{}
	for _, f := range out {
		r, err := f.Reader()
		require.NoError(t, err)
		body, err := io.ReadAll(r)
		r.Close()
		require.NoError(t, err)
		got[f.RelPath()] = string(body)
	}
	require.Equal(t, "home\n", got["hosts"])
	require.Equal(t, "etc\n", got["etc/hosts"])
}

func systemConfig(t *testing.T) *configcue.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "workspaced.cue")
	require.NoError(t, os.WriteFile(path, []byte("package workspaced\n"), 0o644))
	cfg, err := configcue.LoadFilesMode(logging.NewWriterContext(t.Output()), []string{path}, filespine.ModeSystem)
	require.NoError(t, err)
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
	require.Error(t, err, "expected path error")
}
