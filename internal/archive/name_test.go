package archive

import "testing"

func TestIsTarName(t *testing.T) {
	t.Parallel()
	yes := []string{"a.tar", "A.TAR.GZ", "a.tgz", "a.tar.xz", "a.txz", "a.tar.zst", "a.tar.br"}
	for _, name := range yes {
		if !IsTarName(name) {
			t.Fatalf("IsTarName(%q) = false", name)
		}
	}
	if IsTarName("a.zip") || IsTarName("a.sfs") || IsTarName("tool") {
		t.Fatal("non-tar names matched")
	}
}
