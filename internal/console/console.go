package console

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	consolectx "go.datum.net/datumctl/internal/console/context"
)

func Run(ctx context.Context, factory *client.DatumCloudFactory, readOnly bool) error {
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
