package selfupdate

import (
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
		{"go1.27rc1", "1.27.0", true},
		{"go1.28rc1", "1.27.0", true},
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
		[]string{"exec", "go@1.27.0", "--", "build", "-v", "-o", "/tmp/out", "./cmd/modot"},
		viaMise.buildArgs("/tmp/out"))

	viaPath := goToolchain{path: "/usr/local/go/bin/go"}
	require.Equal(t,
		[]string{"build", "-v", "-o", "/tmp/out", "./cmd/modot"},
		viaPath.buildArgs("/tmp/out"))
	require.Equal(t,
		[]string{"list", "-e", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./cmd/modot"},
		viaPath.listArgs())
}

// args must not alias prefix: the builders append to it.
func TestGoToolchainArgsDoNotAliasPrefix(t *testing.T) {
	t.Parallel()
	tc := goToolchain{path: "/bin/mise", prefix: []string{"exec", "go@1.27.0", "--"}}
	_ = tc.buildArgs("/tmp/out")
	_ = tc.listArgs()
	require.Equal(t, []string{"exec", "go@1.27.0", "--"}, tc.prefix)
}
