// Package cmdarg is the workspaced x/cmd field cookbook.
//
// Pick the narrowest Parser that matches the value. StringArg is the leftover
// case (names, refs, URLs, tool specs, remote rsync, free text).
//
//	closed set          cmd.EnumArg[cmdarg.T]
//	existing directory  cmd.WorkDirArg (optional: *cmd.WorkDirArg)
//	existing data dir   cmd.DataDirArg
//	host:port           cmd.AddrArg
//	Go duration         cmd.DurationArg
//	bool switch         cmd.Flag
//	repeatable count    cmd.Count
//	int / float         cmd.IntArg[T] / cmd.FloatArg[T]
//	"--" then rest      cmd.Dash then []T
//	database URL        db.DBArg; Open(ctx); use Value() (never Value().URL() into another opener)
//	opaque token        cmd.StringArg
//
// Do not invent a second opener around DBArg. Off the CLI, db.Open / db.OpenArg.
package cmdarg
