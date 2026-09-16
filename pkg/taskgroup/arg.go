package taskgroup

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
)

// Arg is the root task session. Flatten-embed it so --io/--cpu/--internet
// land on the parent. Zero flags keep the Limits passed to Enter (config
// or DefaultLimits).
type Arg struct {
	IO       cmd.IntArg[int] `long:"io" help:"max concurrent IO tasks (0 keeps config/default)" default:"0"`
	CPU      cmd.IntArg[int] `long:"cpu" help:"max concurrent CPU tasks (0 keeps config/default)" default:"0"`
	Internet cmd.IntArg[int] `long:"internet" help:"max concurrent network tasks (0 keeps config/default)" default:"0"`
}

func (a Arg) Apply(base Limits) Limits {
	if n := a.IO.Value(); n > 0 {
		base.IO = n
	}
	if n := a.CPU.Value(); n > 0 {
		base.CPU = n
	}
	if n := a.Internet.Value(); n > 0 {
		base.Internet = n
	}
	return base
}

func (a Arg) Enter(ctx context.Context, base Limits) (*Session, context.Context) {
	return Enter(ctx, a.Apply(base))
}
