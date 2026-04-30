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
	model := NewAppModel(ctx, factory, consoleCtx)
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}
