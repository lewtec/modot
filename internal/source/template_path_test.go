package source

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTemplatePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want bool
	}{
		{path: "file.tmpl.sh", want: true},
		{path: ".bashrc.tmpl", want: true},
		{path: ".bashrc.d.tmpl/10.sh", want: false},
		{path: ".bashrc.d.tmpl/10.sh.tmpl", want: true},
		{path: ".bashrc.d.tmpl", want: true},
		{path: ".bashrc", want: false},
		{path: ".gitconfig", want: false},
		{path: "plain.txt", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := isTemplatePath(tt.path)
			require.Equal(t, tt.want, got, "isTemplatePath(%q)", tt.path)
		})
	}
}

func TestStripTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want string
	}{
		{path: ".bashrc.tmpl", want: ".bashrc"},
		{path: "file.tmpl.sh", want: "file.sh"},
		{path: "dir/file.tmpl.sh", want: "dir/file.sh"},
		{path: ".bashrc.d.tmpl/10.sh.tmpl", want: ".bashrc.d.tmpl/10.sh"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := stripTemplate(tt.path)
			require.Equal(t, tt.want, got, "stripTemplate(%q)", tt.path)
		})
	}
}
