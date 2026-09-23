package filespine

import "testing"

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
			if got := NamespaceVisible(tt.mode, tt.profile); got != tt.want {
				t.Fatalf("NamespaceVisible(%q, %q) = %v, want %v", tt.mode, tt.profile, got, tt.want)
			}
		})
	}
}

func TestPresetVisible(t *testing.T) {
	t.Parallel()
	if !PresetVisible(ModeSystem, "etc") || !PresetVisible(ModeSystem, "bin") || !PresetVisible(ModeSystem, "root") {
		t.Fatal("system mode should read the system presets")
	}
	if PresetVisible(ModeHome, "etc") {
		t.Fatal("home mode should skip etc")
	}
	if SystemRel("bin") != "usr/local/bin" || SystemRel("etc") != "etc" || SystemRel("root") != "." {
		t.Fatalf("rels bin=%q etc=%q root=%q", SystemRel("bin"), SystemRel("etc"), SystemRel("root"))
	}
}

func TestProfileForTarget(t *testing.T) {
	t.Parallel()
	home := "/home/user"
	got, ok := ProfileForTarget(ModeHome, home, home)
	if !ok || got != ModeHome {
		t.Fatalf("home target = %q %v", got, ok)
	}
	got, ok = ProfileForTarget(ModeSystem, "/", "/")
	if !ok || got != ModeSystem {
		t.Fatalf("system target = %q %v", got, ok)
	}
	if _, ok := ProfileForTarget(ModeHome, "/etc", home); ok {
		t.Fatal("home mode accepted /etc")
	}
	if ApplyDir(ModeHome, "") != "." || ApplyDir("etc", "") != "etc" {
		t.Fatalf("relative home=%q etc=%q", ApplyDir(ModeHome, ""), ApplyDir("etc", ""))
	}
	if ApplyDir("etc", "/") != "/etc" || ApplyDir("bin", "/mnt") != "/mnt/usr/local/bin" {
		t.Fatalf("joined etc=%q bin=%q", ApplyDir("etc", "/"), ApplyDir("bin", "/mnt"))
	}
	if ApplyDir(ModeSystem, "/mnt") != "/mnt" {
		t.Fatalf("system = %q", ApplyDir(ModeSystem, "/mnt"))
	}
}

func TestPrimary(t *testing.T) {
	t.Parallel()
	if got := Primary(""); got != ModeHome {
		t.Fatalf("Primary empty = %q", got)
	}
	if got := Primary(ModeCodebase); got != ModeCodebase {
		t.Fatalf("Primary codebase = %q", got)
	}
	if got := Primary(ModeSystem); got != ModeSystem {
		t.Fatalf("Primary system = %q", got)
	}
}
