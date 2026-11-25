package env_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/driver/env"
	"github.com/stretchr/testify/require"
)

func TestMergeEssentialPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		essential []string
		existing  []string
		want      []string
	}{
		{
			name:      "prepends missing in reverse essential order",
			essential: []string{"A", "B", "C"},
			existing:  []string{"x", "y"},
			want:      []string{"C", "B", "A", "x", "y"},
		},
		{
			name:      "skips essentials already present",
			essential: []string{"A", "B", "C"},
			existing:  []string{"B", "x"},
			want:      []string{"C", "A", "B", "x"},
		},
		{
			name:      "no missing essentials clones existing",
			essential: []string{"A", "B"},
			existing:  []string{"A", "B", "x"},
			want:      []string{"A", "B", "x"},
		},
		{
			name:      "dedupes repeated essentials",
			essential: []string{"A", "A", "B"},
			existing:  []string{"x"},
			want:      []string{"B", "A", "x"},
		},
		{
			name:      "empty essential leaves existing",
			essential: nil,
			existing:  []string{"x"},
			want:      []string{"x"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := env.MergeEssentialPaths(tt.essential, tt.existing)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExpandPathInUsesProvidedHome(t *testing.T) {
	t.Parallel()
	got := env.ExpandPathIn("~/.config/modot", "/data/home")
	want := filepath.Join("/data/home", ".config/modot")
	require.Equal(t, want, got)
}

func TestFindDotfilesRoot(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".dotfiles")
	require.NoError(t, os.Mkdir(root, 0o755))
	got, err := env.FindDotfilesRoot(home)
	require.NoError(t, err)
	require.Equal(t, root, got)
}

func TestEnsureUnderHome(t *testing.T) {
	home := t.TempDir()
	got, err := env.EnsureUnderHome(home, ".local/share/modot")
	require.NoError(t, err)
	want := filepath.Join(home, ".local/share/modot")
	require.Equal(t, want, got)
	_, err = os.Stat(got)
	require.NoError(t, err)
}
