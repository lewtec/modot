package cmdarg

import (
	"os"
	"testing"
)

func TestHomePrefixDefault(t *testing.T) {
	t.Parallel()
	if got := (HomePrefix{}).ArgDefault(); got != "~" {
		t.Fatalf("default = %q", got)
	}
	var prefix HomePrefix
	if err := prefix.Parse(prefix.ArgDefault()); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if prefix.Value() != home {
		t.Fatalf("value = %q", prefix.Value())
	}
}

func TestHomePrefixExpandsTilde(t *testing.T) {
	t.Parallel()
	var prefix HomePrefix
	if err := prefix.Parse("~/stage"); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := home + "/stage"
	if prefix.Value() != want {
		t.Fatalf("value = %q, want %q", prefix.Value(), want)
	}
}

func TestHomePrefixRejectsRelative(t *testing.T) {
	t.Parallel()
	var prefix HomePrefix
	err := prefix.Parse("stage")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSystemPrefixDefault(t *testing.T) {
	t.Parallel()
	if got := (SystemPrefix{}).ArgDefault(); got != "/" {
		t.Fatalf("default = %q", got)
	}
	var prefix SystemPrefix
	if err := prefix.Parse(prefix.ArgDefault()); err != nil {
		t.Fatal(err)
	}
	if prefix.Value() != "/" {
		t.Fatalf("value = %q", prefix.Value())
	}
}

func TestSystemPrefixCleans(t *testing.T) {
	t.Parallel()
	var prefix SystemPrefix
	if err := prefix.Parse("/mnt/"); err != nil {
		t.Fatal(err)
	}
	if prefix.Value() != "/mnt" {
		t.Fatalf("value = %q", prefix.Value())
	}
	err := prefix.Parse("mnt")
	if err == nil {
		t.Fatal("expected error")
	}
}
