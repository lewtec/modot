package main

import (
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	pkg_codebase "github.com/lucasew/workspaced/cmd/workspaced/codebase"
	pkg_daemon "github.com/lucasew/workspaced/cmd/workspaced/daemon"
	pkg_driver "github.com/lucasew/workspaced/cmd/workspaced/driver"
	pkg_experiments "github.com/lucasew/workspaced/cmd/workspaced/experiments"
	pkg_home "github.com/lucasew/workspaced/cmd/workspaced/home"
	pkg_init "github.com/lucasew/workspaced/cmd/workspaced/init"
	pkg_is "github.com/lucasew/workspaced/cmd/workspaced/is"
	pkg_mod "github.com/lucasew/workspaced/cmd/workspaced/mod"
	pkg_open "github.com/lucasew/workspaced/cmd/workspaced/open"
	pkg_selfinstall "github.com/lucasew/workspaced/cmd/workspaced/selfinstall"
	pkg_selfupdate "github.com/lucasew/workspaced/cmd/workspaced/selfupdate"
	pkg_svc "github.com/lucasew/workspaced/cmd/workspaced/svc"
	pkg_system "github.com/lucasew/workspaced/cmd/workspaced/system"
	pkg_tool "github.com/lucasew/workspaced/cmd/workspaced/tool"
	pkg_utils "github.com/lucasew/workspaced/cmd/workspaced/utils"
	"github.com/lucasew/workspaced/pkg/taskgroup"
)

// cli is the workspaced command spec. Process flags live on cmd.App[cli].
type cli struct {
	taskgroup.Arg `flatten:""`
	DryRun        cmd.Flag `short:"d" long:"dry-run" help:"Only show what would be done" ctx:"dry-run"`
	NoCache       cmd.Flag `long:"no-cache" help:"Ignore install/module/source/shell caches; re-fetch locked tools; treat deploy noops as updates (also WORKSPACED_NO_CACHE)" env:"WORKSPACED_NO_CACHE" ctx:"no-cache"`

	Codebase    *pkg_codebase.Command
	Driver      *pkg_driver.Command
	Experiments *pkg_experiments.Command
	Home        *pkg_home.Command
	Init        *pkg_init.Command
	Is          *pkg_is.Command
	Mod         *pkg_mod.Command
	Open        *pkg_open.Command
	Selfinstall *pkg_selfinstall.Command `cmd:"self-install"`
	Selfupdate  *pkg_selfupdate.Command  `cmd:"self-update"`
	Svc         *pkg_svc.Command
	System      *pkg_system.Command
	Tool        *pkg_tool.Command
	Utils       *pkg_utils.Command
	Daemon      *pkg_daemon.Command
}

func (cli) Description() string {
	return "workspaced - declarative user environment manager"
}

func rewriteArgs(args []string) []string {
	return rewriteOpen(args)
}

func rewriteOpen(args []string) []string {
	i := commandIndex(args, "open")
	if i < 0 || i+1 >= len(args) {
		return args
	}
	next := args[i+1]
	if isOption(next) || isOpenSub(next) {
		return args
	}
	out := make([]string, 0, len(args)+1)
	out = append(out, args[:i+1]...)
	out = append(out, "file")
	out = append(out, args[i+1:]...)
	return out
}

func isOpenSub(s string) bool {
	switch s {
	case "file", "webapp", "terminal", "exec", "lazy", "mise":
		return true
	default:
		return false
	}
}

func commandIndex(args []string, name string) int {
	skipVal := false
	for i, a := range args {
		if skipVal {
			skipVal = false
			continue
		}
		if a == "--" {
			return -1
		}
		if isOption(a) {
			if optionTakesValue(a) {
				skipVal = !strings.Contains(a, "=")
			}
			continue
		}
		if a == name {
			return i
		}
		// first non-flag token that isn't open means a different command
		return -1
	}
	return -1
}

func isOption(a string) bool {
	return len(a) > 1 && a[0] == '-' && a != "-"
}

func optionTakesValue(a string) bool {
	name := strings.TrimLeft(a, "-")
	name, _, _ = strings.Cut(name, "=")
	switch name {
	case "profile-dir", "io", "cpu", "internet":
		return true
	default:
		return false
	}
}
