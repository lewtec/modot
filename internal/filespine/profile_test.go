package filespine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamespaceVisible(t *testing.T) {
	t.Parallel()
	tests := []struct {
		mode    string
		profile string
		want    bool
	}{
		{mode: ModeHome, profile: ModeHome, want: true},
		{mode: ModeHome, profile: "etc", want: false},
		{mode: ModeHome, profile: ModeCodebase, want: false},
		{mode: ModeHome, profile: ModeSystem, want: false},
		{mode: "", profile: ModeHome, want: true},
		{mode: ModeCodebase, profile: ModeCodebase, want: true},
		{mode: ModeCodebase, profile: ModeHome, want: false},
		{mode: ModeSystem, profile: ModeSystem, want: true},
		{mode: ModeSystem, profile: "etc", want: false},
		{mode: ModeSystem, profile: ModeHome, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"/"+tt.profile, func(t *testing.T) {
			t.Parallel()
			got := NamespaceVisible(tt.mode, tt.profile)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPresetVisible(t *testing.T) {
	t.Parallel()
	require.True(t, PresetVisible(ModeSystem, "etc") && PresetVisible(ModeSystem, "bin") && PresetVisible(ModeSystem, "root"), "system mode should read the system presets")
	require.False(t, PresetVisible(ModeHome, "etc"), "home mode should skip etc")
	require.Equal(t, "usr/local/bin", SystemRel("bin"))
	require.Equal(t, "etc", SystemRel("etc"))
	require.Equal(t, ".", SystemRel("root"))
}

func TestProfileForTarget(t *testing.T) {
	t.Parallel()
	home := "/home/user"
	got, ok := ProfileForTarget(ModeHome, home, home)
	require.True(t, ok)
	require.Equal(t, ModeHome, got)
	got, ok = ProfileForTarget(ModeSystem, "/", "/")
	require.True(t, ok)
	require.Equal(t, ModeSystem, got)
	_, ok = ProfileForTarget(ModeHome, "/etc", home)
	require.False(t, ok, "home mode accepted /etc")
	require.Equal(t, ".", ApplyDir(ModeHome, ""))
	require.Equal(t, "etc", ApplyDir("etc", ""))
	require.Equal(t, "/etc", ApplyDir("etc", "/"))
	require.Equal(t, "/mnt/usr/local/bin", ApplyDir("bin", "/mnt"))
	require.Equal(t, "/mnt", ApplyDir(ModeSystem, "/mnt"))
}

func TestPrimary(t *testing.T) {
	t.Parallel()
	require.Equal(t, ModeHome, Primary(""))
	require.Equal(t, ModeCodebase, Primary(ModeCodebase))
	require.Equal(t, ModeSystem, Primary(ModeSystem))
}
