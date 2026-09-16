package cmdctx

import (
	"testing"
)

func TestWithNoCache(t *testing.T) {
	ctx := t.Context()
	if IsNoCache(ctx) {
		t.Fatal("default off")
	}
	ctx = WithNoCache(ctx, true)
	if !IsNoCache(ctx) {
		t.Fatal("expected on")
	}
	ctx = WithNoCache(ctx, false)
	if IsNoCache(ctx) {
		t.Fatal("expected off")
	}
}
