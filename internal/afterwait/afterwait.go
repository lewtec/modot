// Package afterwait runs hooks after the task session has finished.
// Register during command Run; call Run from the CLI after progress.Run
// so reports and child processes see real stdio.
package afterwait

import (
	"context"
	"os"
	"os/exec"
	"sync"
)

type ctxKey struct{}

type registry struct {
	mu sync.Mutex
	fn []func() error
}

// With attaches an empty hook list to ctx.
func With(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, &registry{})
}

// Register appends fn to run after the session wait. No-op without With.
func Register(ctx context.Context, fn func() error) {
	if fn == nil {
		return
	}
	r, ok := ctx.Value(ctxKey{}).(*registry)
	if !ok || r == nil {
		return
	}
	r.mu.Lock()
	r.fn = append(r.fn, fn)
	r.mu.Unlock()
}

// Exec registers cmd to run with process stdio after the session wait.
func Exec(ctx context.Context, cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	Register(ctx, func() error {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

// Run executes registered hooks in order. The first error is returned.
func Run(ctx context.Context) error {
	r, ok := ctx.Value(ctxKey{}).(*registry)
	if !ok || r == nil {
		return nil
	}
	r.mu.Lock()
	hooks := r.fn
	r.fn = nil
	r.mu.Unlock()
	var first error
	for _, fn := range hooks {
		if err := fn(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
