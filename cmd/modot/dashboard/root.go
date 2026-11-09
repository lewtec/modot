// Package dashboard serves the config layering dashboard: the discovered
// modot.cue layers, the module schemas injected into the shared options tree,
// and the merged Configuration with the file that supplied each key.
//
// It opens a web view when the machine has a graphical session and serves
// HTTP otherwise. --addr always serves HTTP on that address.
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	lewapp "github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/cmd"
	lewdriver "github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/modot/internal/configdash"
	"github.com/lewtec/modot/internal/configdash/view"
)

// Command is the dashboard: a web view of any layer set, home or codebase.
type Command struct {
	Addr   cmd.StringArg  `default:""         help:"Serve HTTP on this address instead of opening a window" long:"addr"`
	NoOpen cmd.Flag       `                  help:"Serve HTTP on a loopback port and print the URL"     long:"no-open"`
	Open   cmd.Flag       `                  help:"Force the web view even without a graphical session" long:"open"`
	Cwd    cmd.WorkDirArg `                  help:"Directory to start modot.cue discovery from"         long:"cwd"`
	Scope  cmd.StringArg  `default:"codebase" enum:"home,codebase"                                        help:"Layer set to show"                          long:"scope"`
}

func (*Command) Description() string {
	return "Open the config layering dashboard (web view, or HTTP with --addr)"
}

// Run serves the dashboard.
func (c *Command) Run(ctx context.Context) error {
	cwd := c.Cwd.Value()
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if resolved, err := filepath.Abs(cwd); err == nil {
		cwd = resolved
	}

	handler := view.Handler(func(r *http.Request) view.View {
		return loadView(r.Context(), cwd, c.Scope.Value())
	})

	if addr := c.Addr.Value(); addr != "" {
		return serve(ctx, addr, handler)
	}
	if c.NoOpen.Value() {
		return serve(ctx, "127.0.0.1:0", handler)
	}
	if c.Open.Value() || graphical() {
		return openWindow(ctx, handler, cwd)
	}
	return serve(ctx, "127.0.0.1:0", handler)
}

// loadView reloads the config on every request so edits show up without a
// restart. Discovery runs here because the layer set can change at any time.
func loadView(ctx context.Context, cwd, requested string) view.View {
	scope := configdash.Scope(requested)
	if scope != configdash.ScopeHome && scope != configdash.ScopeCodebase {
		scope = configdash.ScopeCodebase
	}
	rep, err := configdash.Load(ctx, scope, cwd)
	if err != nil {
		slog.Warn("dashboard load failed", "cwd", cwd, "scope", scope, "error", err)
		return view.View{LoadErr: err, Partial: &rep}
	}
	return view.View{Report: rep}
}

// graphical reports whether a window can open: an app host owns the surface,
// or the process has a display and a session bus (or a Wayland socket) to
// reach it. A bare DISPLAY with no session is a text-only host.
func graphical() bool {
	if lewdriver.AppMode() {
		return true
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return os.Getenv("DISPLAY") != "" && os.Getenv("DBUS_SESSION_BUS_ADDRESS") != ""
}

func openWindow(ctx context.Context, handler http.Handler, cwd string) error {
	title := "modot config"
	if cwd != "" {
		title = "modot config · " + filepath.Base(cwd)
	}
	return entry.RunAfterView(ctx, func(runCtx context.Context) error {
		return lewapp.App{
			Title:   title,
			Width:   1280,
			Height:  860,
			Handler: lewapp.Web(handler),
		}.Run(runCtx)
	})
}

// serve listens on addr and serves handler until ctx is done.
func serve(ctx context.Context, addr string, handler http.Handler) error {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	url := "http://" + listener.Addr().String()
	slog.Info("dashboard listening", "url", url)
	// stdout is for pipeable records; the URL is the one record here.
	if _, err := fmt.Fprintln(os.Stdout, url); err != nil {
		return err
	}
	server := &http.Server{
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		ReadHeaderTimeout: 10 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		closeErr := server.Shutdown(shutdownCtx)
		slog.Debug("dashboard stopped", "error", closeErr)
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
