package llm

import (
	"bufio"
	"io"
	"strings"
)

// sseEvent is a parsed Server-Sent Event.
type sseEvent struct {
	typ  string // from "event: " line
	data string // from "data: " line
}

// scanSSE reads SSE events from r and sends them on the returned channel.
// The channel is closed when r is exhausted or returns an error.
// Each complete event (delimited by a blank line) is sent as one sseEvent.
func scanSSE(r io.Reader) <-chan sseEvent {
	ch := make(chan sseEvent, 16)
	go func() {
		defer close(ch)
		sc := bufio.NewScanner(r)
		var ev sseEvent
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				// Blank line = event boundary.
				if ev.data != "" || ev.typ != "" {
					ch <- ev
					ev = sseEvent{}
				}
			case strings.HasPrefix(line, "event:"):
				ev.typ = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				ev.data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
		// Flush any trailing event not followed by a blank line.
		if ev.data != "" || ev.typ != "" {
			ch <- ev
		}
	}()
	return ch
}
