package tuictx

import (
	"time"

	"go.datum.net/datumctl/internal/datumconfig"
)

// TUIContext holds the display-ready context fields for the TUI header.
type TUIContext struct {
	UserEmail       string
	UserName        string
	OrgName         string
	ProjectName     string
	Namespace       string
	ActiveCtx       *datumconfig.DiscoveredContext
	Config          *datumconfig.ConfigV1Beta1
	ActivePaneLabel string    // "NAV" | "TABLE" | "DETAIL"
	LastRefresh     time.Time // zero value = never refreshed
	ResourceCount   int       // rows in current view
	CurrentType     string    // selected resource type name
	ReadOnly        bool      // when true, mutation operations are disabled
	Refreshing      bool      // true while a manual refresh is in flight
	SpinnerFrame    string    // current spinner frame, set on each TickMsg
}

// FromConfig builds a TUIContext from the loaded config and its current context entry.
func FromConfig(cfg *datumconfig.ConfigV1Beta1) TUIContext {
	if cfg == nil {
		return TUIContext{}
	}

	tc := TUIContext{Config: cfg}

	ctx := cfg.CurrentContextEntry()
	if ctx == nil {
		return tc
	}
	tc.ActiveCtx = ctx
	tc.Namespace = ctx.Namespace

	session := cfg.ActiveSessionEntry()
	if session != nil {
		tc.UserEmail = session.UserEmail
		tc.UserName = session.UserName
	}

	tc.OrgName = cfg.OrgDisplayName(ctx.OrganizationID)

	if ctx.ProjectID != "" {
		tc.ProjectName = cfg.ProjectDisplayName(ctx.ProjectID)
	}

	return tc
}
