package selfupdate

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoVersionAtLeast(t *testing.T) {
	t.Parallel()
	cases := []struct {
		version string
		want    string
		ok      bool
	}{
		{"go1.27.0", "1.27.0", true},
		{"go1.28.3", "1.27.0", true},
		{"go2.0.0", "1.27.0", true},
		{"go1.26.9", "1.27.0", false},
		{"go1.27.0", "1.28.0", false},
		{"go1.27rc1", "1.27.0", false},
		{"go1.28rc1", "1.27.0", false},
		{"go1.27devel", "1.27.0", true},
	}
	for _, tc := range cases {
		require.Equal(t, tc.ok, goVersionAtLeast(tc.version, tc.want), "%s >= %s", tc.version, tc.want)
	}
}

// The `go build -o <path>` argv must never leak flags to mise: they always come
// after a `--`, so mise cannot read `-o` as its own option (the Apple Silicon
// "--env <ENV>" failure came from that).
func TestGoToolchainKeepsGoFlagsAfterDoubleDash(t *testing.T) {
	t.Parallel()
	viaMise := goToolchain{path: "/bin/mise", prefix: []string{"exec", "go@1.27.0", "--"}}
	require.Equal(t,
		[]string{"exec", "go@1.27.0", "--", "go", "build", "-v", "-o", "/tmp/out", "./cmd/modot"},
		viaMise.buildArgs("/tmp/out"))
	require.Equal(t, []string{"exec", "go@1.27.0", "--", "go", "env", "GOVERSION"}, viaMise.versionArgs())

	viaPath := goToolchain{path: "/usr/local/go/bin/go"}
	require.Equal(t,
		[]string{"build", "-v", "-o", "/tmp/out", "./cmd/modot"},
		viaPath.buildArgs("/tmp/out"))
	require.Equal(t,
		[]string{"list", "-e", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./cmd/modot"},
		viaPath.listArgs())
	require.Equal(t, []string{"env", "GOVERSION"}, viaPath.versionArgs())
}

// A mise prefix needs `go` named after its `--`: only a program may start
// there, and a bare `NAME=value` would be run as a program.
func TestGoToolchainPinnedStillRunsGo(t *testing.T) {
	t.Parallel()
	viaMise := goToolchain{
		path:   "/bin/mise",
		prefix: []string{"exec", "go@1.27.0", "--"},
		pin:    pinLocal,
	}
	require.Equal(t,
		[]string{"exec", "go@1.27.0", "--", "go", "build", "-v", "-o", "/tmp/out", "./cmd/modot"},
		viaMise.buildArgs("/tmp/out"))
	require.Equal(t,
		[]string{"exec", "go@1.27.0", "--", "go", "env", "GOVERSION"},
		viaMise.versionArgs())

	viaPath := goToolchain{path: "/usr/local/go/bin/go", pin: pinAuto}
	require.Equal(t, []string{"env", "GOVERSION"}, viaPath.versionArgs())
	require.Equal(t, []string{"build", "-v", "-o", "/tmp/out", "./cmd/modot"}, viaPath.buildArgs("/tmp/out"))
}

// GOTOOLCHAIN travels in the environment, so it reaches mise and go alike.
func TestGoToolchainPinsGotoolchainInEnv(t *testing.T) {
	t.Parallel()
	tc := goToolchain{path: "/usr/local/go/bin/go", version: "go1.25.1", pin: pinAuto}
	base := []string{"PATH=/usr/bin"}
	require.Equal(t, []string{"PATH=/usr/bin", "GOTOOLCHAIN=auto"}, tc.env(base))
	require.Equal(t, []string{"PATH=/usr/bin"}, base, "env must not modify the base slice")
	require.NotContains(t, tc.buildArgs("/tmp/out"), "GOTOOLCHAIN=auto")

	unpinned := goToolchain{path: "/usr/local/go/bin/go"}
	require.Equal(t, base, unpinned.env(base))
}

// The exec driver leaves Cmd.Env nil (inherit). Pinning must not turn that into
// a go that has no HOME, PATH or GOPATH.
func TestGoToolchainEnvKeepsInheritedEnvironment(t *testing.T) {
	t.Setenv("MODOT_TEST_INHERITED", "1")
	for _, tc := range []goToolchain{{path: "/usr/local/go/bin/go"}, {path: "/usr/local/go/bin/go", pin: pinLocal}} {
		env := tc.env(nil)
		require.Contains(t, env, "MODOT_TEST_INHERITED=1")
		require.Contains(t, strings.Join(env, "\n"), "\nHOME=", "the inherited environment is gone")
	}
}

// A mise prefix that lost its `--` would hand `-o` to mise as its own flag.
func TestGoToolchainMisePrefixAlwaysEndsWithDoubleDash(t *testing.T) {
	t.Parallel()
	for _, pin := range []string{"", pinLocal} {
		tc := goToolchain{path: "/bin/mise", prefix: []string{"exec", "go@1.27.0", "--"}, pin: pin}
		args := tc.buildArgs("/tmp/out")
		dash := slices.Index(args, "--")
		require.Greater(t, dash, 0, "prefix %v has no --", args)
		require.Contains(t, args[dash+1:], "build")
		require.NotContains(t, slices.Clone(args[:dash]), "-o")
	}
}

func TestGoSupportsToolchainSwitch(t *testing.T) {
	t.Parallel()
	require.True(t, goSupportsToolchainSwitch(goToolchain{version: "go1.21.0"}))
	require.True(t, goSupportsToolchainSwitch(goToolchain{version: "go1.25.1"}))
	require.False(t, goSupportsToolchainSwitch(goToolchain{version: "go1.20.14"}))
}

// args must not alias prefix: the builders append to it.
func TestGoToolchainArgsDoNotAliasPrefix(t *testing.T) {
	t.Parallel()
	tc := goToolchain{path: "/bin/mise", prefix: []string{"exec", "go@1.27.0", "--"}, pin: pinLocal}
	_ = tc.buildArgs("/tmp/out")
	_ = tc.listArgs()
	require.Equal(t, []string{"exec", "go@1.27.0", "--"}, tc.prefix)
}
