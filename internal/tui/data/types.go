package data

import "time"

// EventRow is a single Kubernetes event row for display in the DetailPane events table.
// Age is computed from LastTimestamp, falling back to EventTime when LastTimestamp is zero. // AC#1
type EventRow struct {
	Type          string    // "Normal" or "Warning"
	Reason        string    // e.g., "FailedScheduling", "BackOff", "SuccessfulCreate"
	Message       string    // human-readable event message
	Count         int32     // aggregated count; 0 = single / unknown occurrence
	LastTimestamp time.Time // used to compute Age; falls back to EventTime when zero
	EventTime     time.Time // new-style event timestamp; used when LastTimestamp is zero
}

type LoadState int

const (
	LoadStateIdle LoadState = iota
	LoadStateLoading
	LoadStateError
)

type AllowanceBucket struct {
	Name                  string
	ConsumerKind          string // quota.miloapis.com/consumer-kind label
	ConsumerName          string // quota.miloapis.com/consumer-name label
	ResourceType          string // spec.resourceType
	Limit                 int64
	Allocated             int64
	Available             int64
	ClaimCount            int
	ContributingGrantRefs []string
	LastReconciliation    time.Time
}

type ResourceType struct {
	Name        string
	Kind        string
	Group       string
	Version     string
	Namespaced  bool
	Description string // from OpenAPI v3 schema; empty if unavailable
}

type ResourceRow struct {
	Name      string   // from object metadata, used for describe
	Namespace string   // from object metadata
	Cells     []string // display cells matching the column definitions
}

type WatchEvent struct {
	Type string
	Row  ResourceRow
}
