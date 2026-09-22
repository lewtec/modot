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
		{mode: ModeHome, profile: "etc", want: true},
		{mode: ModeHome, profile: "bin", want: true},
		{mode: ModeHome, profile: ModeCodebase, want: false},
		{mode: ModeHome, profile: ModeSystem, want: false},
		{mode: "", profile: ModeHome, want: true},
		{mode: "", profile: ModeCodebase, want: false},
		{mode: ModeCodebase, profile: ModeCodebase, want: true},
		{mode: ModeCodebase, profile: ModeHome, want: false},
		{mode: ModeSystem, profile: ModeSystem, want: true},
		{mode: ModeSystem, profile: "etc", want: false},
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
