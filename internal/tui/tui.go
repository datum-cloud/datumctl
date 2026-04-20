package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	tuictx "go.datum.net/datumctl/internal/tui/context"
)

func Run(ctx context.Context, factory *client.DatumCloudFactory, readOnly bool) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	tuiCtx := tuictx.FromConfig(cfg)
	tuiCtx.ReadOnly = readOnly
	model := NewAppModel(ctx, factory, tuiCtx)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}
