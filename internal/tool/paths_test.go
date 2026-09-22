package tool

import (
	"path/filepath"
	"testing"
)

func TestBinaryCandidates(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		cmdName string
		want    []string
	}{
		{
			name:    "standard layout",
			baseDir: "/tools/github-cli-cli/2.0.0",
			cmdName: "gh",
			want: []string{
				filepath.Join("/tools/github-cli-cli/2.0.0", "bin", "gh"),
				filepath.Join("/tools/github-cli-cli/2.0.0", "bin", "gh.exe"),
				filepath.Join("/tools/github-cli-cli/2.0.0", "bin", "gh.cmd"),
				filepath.Join("/tools/github-cli-cli/2.0.0", "bin", "gh.bat"),
				filepath.Join("/tools/github-cli-cli/2.0.0", "gh"),
				filepath.Join("/tools/github-cli-cli/2.0.0", "gh.exe"),
				filepath.Join("/tools/github-cli-cli/2.0.0", "gh.cmd"),
				filepath.Join("/tools/github-cli-cli/2.0.0", "gh.bat"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BinaryCandidates(tt.baseDir, tt.cmdName)
			if len(got) != len(tt.want) {
				t.Fatalf("len(BinaryCandidates) = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BinaryCandidates[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
