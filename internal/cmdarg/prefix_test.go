package cmdarg

import (
	"os"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
)

func TestPrefixExpandsTilde(t *testing.T) {
	t.Parallel()
	var prefix Prefix
	if err := prefix.Parse("~"); err != nil {
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

func TestPrefixAcceptsSlashAndDot(t *testing.T) {
	t.Parallel()
	var root Prefix
	if err := root.Parse("/"); err != nil {
		t.Fatal(err)
	}
	if root.Value() != "/" {
		t.Fatalf("root = %q", root.Value())
	}
	var here Prefix
	if err := here.Parse("."); err != nil {
		t.Fatal(err)
	}
	if here.Value() != "." {
		t.Fatalf("dot = %q", here.Value())
	}
}

func TestPrefixFieldDefaults(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	type homeArgs struct {
		Prefix Prefix `long:"prefix" default:"~"`
	}
	parsedHome, err := cmd.Parse[homeArgs]()
	if err != nil {
		t.Fatal(err)
	}
	if parsedHome.Prefix.Value() != home {
		t.Fatalf("home default = %q", parsedHome.Prefix.Value())
	}

	type dotArgs struct {
		Prefix Prefix `long:"prefix" default:"."`
	}
	parsedDot, err := cmd.Parse[dotArgs]()
	if err != nil {
		t.Fatal(err)
	}
	if parsedDot.Prefix.Value() != "." {
		t.Fatalf("codebase default = %q", parsedDot.Prefix.Value())
	}

	type rootArgs struct {
		Prefix Prefix `long:"prefix" default:"/"`
	}
	parsedRoot, err := cmd.Parse[rootArgs]()
	if err != nil {
		t.Fatal(err)
	}
	if parsedRoot.Prefix.Value() != "/" {
		t.Fatalf("system default = %q", parsedRoot.Prefix.Value())
	}
}

func TestPrefixRejectsMissing(t *testing.T) {
	t.Parallel()
	var prefix Prefix
	if err := prefix.Parse(t.TempDir() + "/missing"); err == nil {
		t.Fatal("expected error")
	}
}
