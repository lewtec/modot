package filespine

import "testing"

func TestIsTemplatePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want bool
	}{
		{path: "file.tmpl.sh", want: true},
		{path: ".bashrc.tmpl", want: true},
		{path: ".bashrc.d.tmpl/10.sh", want: true},
		{path: ".bashrc.d.tmpl", want: true},
		{path: ".bashrc", want: false},
		{path: ".gitconfig", want: false},
		{path: "plain.txt", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := IsTemplatePath(tt.path); got != tt.want {
				t.Fatalf("IsTemplatePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
