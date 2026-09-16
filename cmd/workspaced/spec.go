package main

import (
	"log/slog"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
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
)

// cli is the workspaced command spec. Process flags live on cmd.App[cli].
type cli struct {
	taskgroup.Arg `flatten:"" ctx:"taskgroup"`
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

// Setup runs after cmd.App replaces slog.Default with a TextHandler.
func (*cli) Setup() error {
	if processLogger != nil {
		slog.SetDefault(processLogger)
	}
	return nil
}
