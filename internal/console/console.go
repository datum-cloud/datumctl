package console

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"k8s.io/klog/v2"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	consolectx "go.datum.net/datumctl/internal/console/context"
)

func Run(ctx context.Context, factory *client.DatumCloudFactory, readOnly bool) error {
	// Redirect klog (used by client-go) to a log file so its output doesn't
	// corrupt the TUI. Logs land at $(os.UserCacheDir)/datumctl/console.log.
	closeLog := redirectKlogToFile()
	defer closeLog()

	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	consoleCtx := consolectx.FromConfig(cfg)
	consoleCtx.ReadOnly = readOnly

	// Derive the auth hostname from the active session so the in-TUI login
	// overlay contacts the same endpoint the user previously authenticated
	// against (e.g. staging). Fall back to the canonical production hostname.
	authHostname := "auth.datum.net"
	if s := cfg.ActiveSessionEntry(); s != nil && s.Endpoint.AuthHostname != "" {
		authHostname = s.Endpoint.AuthHostname
	}

	model := NewAppModel(ctx, factory, consoleCtx, authHostname)
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}

func redirectKlogToFile() (close func()) {
	f, err := openConsoleLog()
	if err != nil {
		klog.SetLogger(logr.Discard())
		return func() {}
	}
	klog.SetLogger(funcr.New(func(prefix, args string) {
		fmt.Fprintln(f, prefix, args)
	}, funcr.Options{}))
	return func() { f.Close() }
}

func openConsoleLog() (*os.File, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(cacheDir, "datumctl")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(logDir, "console.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}
