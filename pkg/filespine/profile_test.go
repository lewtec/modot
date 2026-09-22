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
		{mode: ModeHome, profile: "bin", want: false},
		{mode: ModeHome, profile: ModeCodebase, want: false},
		{mode: ModeHome, profile: ModeSystem, want: false},
		{mode: "", profile: ModeHome, want: true},
		{mode: "", profile: "etc", want: false},
		{mode: "", profile: ModeCodebase, want: false},
		{mode: ModeCodebase, profile: ModeCodebase, want: true},
		{mode: ModeCodebase, profile: ModeHome, want: false},
		{mode: ModeSystem, profile: ModeSystem, want: true},
		{mode: ModeSystem, profile: "etc", want: true},
		{mode: ModeSystem, profile: "bin", want: true},
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

func TestProfileForTarget(t *testing.T) {
	t.Parallel()
	home := "/home/user"
	got, ok := ProfileForTarget(ModeHome, home, home)
	if !ok || got != ModeHome {
		t.Fatalf("home target = %q %v", got, ok)
	}
	got, ok = ProfileForTarget(ModeSystem, "/etc", "/")
	if !ok || got != "etc" {
		t.Fatalf("etc target = %q %v", got, ok)
	}
	got, ok = ProfileForTarget(ModeSystem, "/mnt/etc", "/mnt")
	if !ok || got != "etc" {
		t.Fatalf("prefixed etc target = %q %v", got, ok)
	}
	if _, ok := ProfileForTarget(ModeHome, "/etc", home); ok {
		t.Fatal("home mode accepted /etc")
	}
	if _, ok := ProfileForTarget(ModeCodebase, "/etc", "/repo"); ok {
		t.Fatal("codebase mode accepted /etc")
	}
	if got, ok := ProfileForTarget(ModeHome, "", home); !ok || got != ModeHome {
		t.Fatalf("empty target = %q %v", got, ok)
	}
	if RelDir("etc") != "etc" || RelDir("bin") != "usr/local/bin" || RelDir(ModeHome) != "." {
		t.Fatalf("RelDir etc=%q bin=%q home=%q", RelDir("etc"), RelDir("bin"), RelDir(ModeHome))
	}
	if ApplyDir("etc", "") != "etc" {
		t.Fatalf("ApplyDir etc relative = %q", ApplyDir("etc", ""))
	}
	if ApplyDir("etc", "/") != "/etc" || ApplyDir("etc", "/mnt") != "/mnt/etc" {
		t.Fatalf("ApplyDir etc = %q / %q", ApplyDir("etc", "/"), ApplyDir("etc", "/mnt"))
	}
	if ApplyDir("bin", "/") != "/usr/local/bin" {
		t.Fatalf("ApplyDir bin = %q", ApplyDir("bin", "/"))
	}
	if ApplyDir("root", "/mnt") != "/mnt" || ApplyDir(ModeSystem, "/mnt") != "/mnt" {
		t.Fatalf("ApplyDir root = %q system = %q", ApplyDir("root", "/mnt"), ApplyDir(ModeSystem, "/mnt"))
	}
	if ApplyDir(ModeHome, home) != home {
		t.Fatalf("ApplyDir home = %q", ApplyDir(ModeHome, home))
	}
	if got := Visible(ModeCodebase); len(got) != 1 || got[0] != ModeCodebase {
		t.Fatalf("Visible codebase = %v", got)
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
