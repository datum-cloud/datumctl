package data

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// EventsLoadedMsg is the result message returned by LoadEventsCmd. // AC#27
type EventsLoadedMsg struct {
	Events []EventRow
	Err    error
}

// LoadEventsCmd dispatches a ListEvents request and returns the result as an EventsLoadedMsg. // AC#27
func LoadEventsCmd(
	ctx                     context.Context,
	rc                      ResourceClient,
	involvedObjectKind      string,
	involvedObjectName      string,
	involvedObjectNamespace string,
) tea.Cmd {
	return func() tea.Msg {
		events, err := rc.ListEvents(ctx, involvedObjectKind, involvedObjectName, involvedObjectNamespace)
		return EventsLoadedMsg{Events: events, Err: err}
	}
}
