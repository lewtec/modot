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
	t.Cleanup(func() { _ = prefix.Value().Close() })
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if prefix.Value().Name() != home {
		t.Fatalf("name = %q", prefix.Value().Name())
	}
	if _, err := prefix.Value().Stat("."); err != nil {
		t.Fatal(err)
	}
}

func TestHomePrefixExpandsTilde(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var prefix HomePrefix
	if err := prefix.Parse(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = prefix.Value().Close() })
	if prefix.Value().Name() != dir {
		t.Fatalf("name = %q", prefix.Value().Name())
	}
}

func TestHomePrefixRejectsRelative(t *testing.T) {
	t.Parallel()
	var prefix HomePrefix
	if err := prefix.Parse("stage"); err == nil {
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
	t.Cleanup(func() { _ = prefix.Value().Close() })
	if prefix.Value().Name() != "/" {
		t.Fatalf("name = %q", prefix.Value().Name())
	}
	if _, err := prefix.Value().Stat("."); err != nil {
		t.Fatal(err)
	}
}

func TestSystemPrefixCleans(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var prefix SystemPrefix
	if err := prefix.Parse(dir + "/"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = prefix.Value().Close() })
	if prefix.Value().Name() != dir {
		t.Fatalf("name = %q", prefix.Value().Name())
	}
	if err := prefix.Parse("mnt"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPrefixMissingDirectory(t *testing.T) {
	t.Parallel()
	var prefix SystemPrefix
	if err := prefix.Parse(t.TempDir() + "/missing"); err == nil {
		t.Fatal("expected error")
	}
}
