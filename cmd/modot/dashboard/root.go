// Package dashboard serves the config layering dashboard: the discovered
// modot.cue layers, the module schemas injected into the shared options tree,
// and the merged Configuration with the file that supplied each key.
//
// It opens a web view when the machine has a graphical session and serves
// HTTP otherwise. --addr always serves HTTP on that address. Home and codebase
// layers are both served, each under its own path (/home, /codebase); the
// window opens on home when that scope resolves.
package dashboard

import (
	"context"
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

const (
	// windowWidth and windowHeight are the web view's initial size.
	windowWidth  = 1280
	windowHeight = 860
	// readHeaderTimeout bounds how long a client may take to send headers.
	readHeaderTimeout = 10 * time.Second
	// shutdownGrace lets in-flight requests finish on shutdown.
	shutdownGrace = 2 * time.Second
)

// Command is the dashboard: a web view of any layer set, home or codebase.
//
//nolint:tagalign // struct tags cannot wrap.
type Command struct {
	Addr cmd.StringArg  `default:"" help:"Serve HTTP on this address instead of opening a window" long:"addr"`
	Cwd  cmd.WorkDirArg `help:"Directory to start modot.cue discovery from" long:"cwd"`
}

// Description names the command in help output.
func (*Command) Description() string {
	return "Open the config layering dashboard (web view, or HTTP with --addr)"
}

// Run serves the dashboard.
func (c *Command) Run(ctx context.Context) error {
	cwd, err := workingDir(c.Cwd.Value())
	if err != nil {
		return err
	}

	reload := func(r *http.Request, scope configdash.Scope) view.View {
		return loadView(r.Context(), cwd, scope)
	}
	handler := view.Handler(landingScope(ctx, cwd), reload)

	if addr := c.Addr.Value(); addr != "" {
		return serve(ctx, addr, handler)
	}
	if graphical() {
		return openWindow(ctx, handler, cwd)
	}

	return serve(ctx, "127.0.0.1:0", handler)
}

// workingDir resolves the directory discovery starts from.
func workingDir(cwd string) (string, error) {
	if cwd == "" {
		resolved, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}

		cwd = resolved
	}
	absolute, err := filepath.Abs(cwd)
	if err != nil {
		// Abs only fails when the process cannot ask for its own directory;
		// discovery handles a relative path the same way.
		return cwd, nil //nolint:nilerr // the relative path is still usable
	}

	return absolute, nil
}

// landingScope picks the scope "/" redirects to: home layers when discovery
// finds them, codebase next, then system. Every route is served either way, so
// this only decides which one the window opens on.
func landingScope(ctx context.Context, cwd string) configdash.Scope {
	homeErr := discoverable(ctx, configdash.ScopeHome, cwd)
	if homeErr == nil {
		return configdash.ScopeHome
	}
	codebaseErr := discoverable(ctx, configdash.ScopeCodebase, cwd)
	if codebaseErr == nil {
		return configdash.ScopeCodebase
	}
	systemErr := discoverable(ctx, configdash.ScopeSystem, cwd)
	if systemErr == nil {
		return configdash.ScopeSystem
	}
	slog.Warn("dashboard: no layers for any scope", "cwd", cwd, "home", homeErr,
		"codebase", codebaseErr, "system", systemErr)

	return configdash.ScopeHome
}

// discoverable reports whether scope finds at least one layer.
func discoverable(ctx context.Context, scope configdash.Scope, cwd string) error {
	_, err := configdash.Discover(ctx, scope, cwd)
	if err == nil {
		return nil
	}

	return fmt.Errorf("discover %s layers: %w", scope, err)
}

// loadView reloads the config on every request so edits show up without a
// restart. Discovery runs here because the layer set can change at any time.
func loadView(ctx context.Context, cwd string, scope configdash.Scope) view.View {
	rep, err := configdash.Load(ctx, scope, cwd)
	if err != nil {
		slog.Warn("dashboard load failed", "cwd", cwd, "scope", scope, "error", err)

		return view.View{Report: configdash.Report{
			Scope: scope, Mode: "", Root: "", Layers: nil, Modules: nil, Merged: nil,
			MergedErr: nil, Stats: configdash.Stats{Keys: 0, FromModules: 0, FromRoot: 0, Conflicted: 0},
			Sources: nil,
		}, Theme: "", LoadErr: err, Partial: &rep, ModuleName: "", Path: "", Scope: scope}
	}

	return view.View{Report: rep, Theme: "", LoadErr: nil, Partial: nil,
		ModuleName: "", Path: "", Scope: scope}
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

// openWindow runs the dashboard in a lewkit web view.
func openWindow(ctx context.Context, handler http.Handler, cwd string) error {
	title := "modot config"
	if cwd != "" {
		title = "modot config · " + filepath.Base(cwd)
	}

	runErr := entry.RunAfterView(ctx, func(runCtx context.Context) error {
		appErr := lewapp.App{
			Title:   title,
			Width:   windowWidth,
			Height:  windowHeight,
			Handler: lewapp.Web(handler),
			NoUI:    false,
		}.Run(runCtx)
		if appErr != nil {
			return fmt.Errorf("run web view: %w", appErr)
		}

		return nil
	})
	if runErr != nil {
		return fmt.Errorf("open web view: %w", runErr)
	}

	return nil
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
	_, err = fmt.Fprintln(os.Stdout, url)
	if err != nil {
		return fmt.Errorf("print url: %w", err)
	}

	server := &http.Server{
		Addr:                         "",
		Handler:                      handler,
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    nil,
		ReadTimeout:                  0,
		ReadHeaderTimeout:            readHeaderTimeout,
		WriteTimeout:                 0,
		IdleTimeout:                  0,
		MaxHeaderBytes:               0,
		MaxHeaderValueCount:          0,
		TLSNextProto:                 nil,
		ConnState:                    nil,
		ErrorLog:                     nil,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		ConnContext:           nil,
		HTTP2:                 nil,
		Protocols:             nil,
		DisableClientPriority: false,
	}
	serveErr := make(chan error, 1)

	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownGrace)
		defer cancel()

		closeErr := server.Shutdown(shutdownCtx)
		slog.Debug("dashboard stopped", "error", closeErr)

		return nil
	case err := <-serveErr:
		return fmt.Errorf("serve dashboard: %w", err)
	}
}
