package main

import (
	"log/slog"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	pkg_codebase "github.com/lewtec/modot/cmd/modot/codebase"
	pkg_daemon "github.com/lewtec/modot/cmd/modot/daemon"
	pkg_driver "github.com/lewtec/modot/cmd/modot/driver"
	pkg_experiments "github.com/lewtec/modot/cmd/modot/experiments"
	pkg_home "github.com/lewtec/modot/cmd/modot/home"
	pkg_init "github.com/lewtec/modot/cmd/modot/init"
	pkg_is "github.com/lewtec/modot/cmd/modot/is"
	pkg_mod "github.com/lewtec/modot/cmd/modot/mod"
	pkg_open "github.com/lewtec/modot/cmd/modot/open"
	pkg_selfinstall "github.com/lewtec/modot/cmd/modot/selfinstall"
	pkg_selfupdate "github.com/lewtec/modot/cmd/modot/selfupdate"
	pkg_svc "github.com/lewtec/modot/cmd/modot/svc"
	pkg_system "github.com/lewtec/modot/cmd/modot/system"
	pkg_tool "github.com/lewtec/modot/cmd/modot/tool"
	pkg_utils "github.com/lewtec/modot/cmd/modot/utils"
)

// cli is the modot command spec. Process flags live on cmd.App[cli].
type cli struct {
	taskgroup.Arg `flatten:""`
	DryRun        cmd.Flag `short:"d" long:"dry-run" help:"Only show what would be done" ctx:"dry-run"`
	NoCache       cmd.Flag `long:"no-cache" help:"Ignore install/module/source/shell caches; re-fetch locked tools; treat deploy noops as updates (also MODOT_NO_CACHE)" env:"MODOT_NO_CACHE" ctx:"no-cache"`

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
	return "modular dotfiles"
}

// Setup runs after cmd.App replaces slog.Default with x/logging.NewHandler
// on os.Stderr. Always restore: App.Setup installs that handler on every call.
func (*cli) Setup() error {
	if processLogger != nil {
		slog.SetDefault(processLogger)
	}
	return nil
}
