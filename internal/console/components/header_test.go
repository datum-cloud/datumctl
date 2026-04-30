package components

import (
	"strings"
	"testing"
	"time"

	tuictx "go.datum.net/datumctl/internal/console/context"
)

func TestHeaderModel_View(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		ctx          tuictx.TUIContext
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "Refreshing=true shows refreshing label and hides updated label",
			ctx: tuictx.TUIContext{
				Refreshing:   true,
				SpinnerFrame: "⣷",
				LastRefresh:  time.Now().Add(-30 * time.Second),
			},
			wantContains: []string{"refreshing…"},
			wantAbsent:   []string{"updated"},
		},
		{
			name: "Refreshing=true with empty SpinnerFrame still shows refreshing label",
			ctx: tuictx.TUIContext{
				Refreshing: true,
			},
			wantContains: []string{"refreshing…"},
			wantAbsent:   []string{"updated"},
		},
		{
			name: "Refreshing=false with LastRefresh shows updated label",
			ctx: tuictx.TUIContext{
				Refreshing:  false,
				LastRefresh: time.Now().Add(-30 * time.Second),
			},
			wantContains: []string{"updated"},
			wantAbsent:   []string{"refreshing…"},
		},
		{
			name:         "Refreshing=false and zero LastRefresh shows neither label",
			ctx:          tuictx.TUIContext{},
			wantAbsent:   []string{"refreshing…", "updated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewHeaderModel(tt.ctx)
			h.Width = 80
			got := stripANSI(h.View())
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("View() missing %q\ngot:\n%s", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("View() should not contain %q\ngot:\n%s", absent, got)
				}
			}
		})
	}
}
