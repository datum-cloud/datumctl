package data

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// HistoryLoadedMsg carries the result of a LoadHistory call.
type HistoryLoadedMsg struct {
	// Identity tuple so AppModel can verify the response matches the current target.
	APIGroup  string
	Kind      string
	Name      string
	Namespace string

	Rows      []HistoryRow
	Manifests []map[string]any
	Truncated bool
	Err       error
	Unauthorized bool
}

// LoadHistoryCmd dispatches a history fetch for the given resource.
func LoadHistoryCmd(ctx context.Context, hc *HistoryClient, rt ResourceType, name, namespace string) tea.Cmd {
	return func() tea.Msg {
		rows, manifests, truncated, err := hc.LoadHistory(ctx, rt, name, namespace)
		if err != nil {
			return HistoryLoadedMsg{
				APIGroup:     rt.Group,
				Kind:         rt.Kind,
				Name:         name,
				Namespace:    namespace,
				Err:          err,
				Unauthorized: hc.IsUnauthorized(err),
			}
		}
		return HistoryLoadedMsg{
			APIGroup:  rt.Group,
			Kind:      rt.Kind,
			Name:      name,
			Namespace: namespace,
			Rows:      rows,
			Manifests: manifests,
			Truncated: truncated,
		}
	}
}
