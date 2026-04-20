package data

import (
	"sort"
	"strings"
)

// PlatformHealthSummary is the right-block projection over a slice of
// AllowanceBuckets for the active consumer. Pure/deterministic — no side
// effects, no I/O.
type PlatformHealthSummary struct {
	TotalGovernedTypes int
	ConstrainedTypes   int
	TopThree           []ConstrainedRow
}

// ConstrainedRow describes a single governed resource type for the landing
// right-block. PercentInt is an integer floor of (Allocated / Limit * 100).
type ConstrainedRow struct {
	ResourceType string
	Label        string
	Allocated    int64
	Limit        int64
	PercentInt   int
	Near         bool
	Full         bool
}

// ComputePlatformHealthSummary returns the landing right-block summary for the
// active consumer. See FB-015 §3a for the algorithm.
//
// Inputs:
//   - buckets: flat list as returned by BucketClient.ListAllowanceBuckets
//   - activeKind/activeName: the active consumer (Project or Organization)
//   - registrations: optional; used only for label resolution via
//     ResolveDescription
func ComputePlatformHealthSummary(
	buckets []AllowanceBucket,
	activeKind, activeName string,
	registrations []ResourceRegistration,
) PlatformHealthSummary {
	if activeKind == "" || len(buckets) == 0 {
		return PlatformHealthSummary{}
	}

	type agg struct {
		rt        string
		group     string
		name      string
		allocated int64
		limit     int64
		util      float64
	}
	byType := map[string]*agg{}
	for _, b := range buckets {
		if !strings.EqualFold(b.ConsumerKind, activeKind) || b.ConsumerName != activeName {
			continue
		}
		if b.Limit <= 0 {
			continue
		}
		util := float64(b.Allocated) / float64(b.Limit)
		existing, ok := byType[b.ResourceType]
		if !ok {
			group, name := SplitResourceType(b.ResourceType)
			byType[b.ResourceType] = &agg{
				rt:        b.ResourceType,
				group:     group,
				name:      name,
				allocated: b.Allocated,
				limit:     b.Limit,
				util:      util,
			}
			continue
		}
		if util > existing.util {
			existing.allocated = b.Allocated
			existing.limit = b.Limit
			existing.util = util
		}
	}

	if len(byType) == 0 {
		return PlatformHealthSummary{}
	}

	aggs := make([]*agg, 0, len(byType))
	for _, a := range byType {
		aggs = append(aggs, a)
	}

	rows := make([]ConstrainedRow, len(aggs))
	for i, a := range aggs {
		label := ResolveDescription(registrations, a.group, a.name)
		if label == "" {
			label = a.name
		}
		pctInt := int(a.util * 100)
		rows[i] = ConstrainedRow{
			ResourceType: a.rt,
			Label:        label,
			Allocated:    a.allocated,
			Limit:        a.limit,
			PercentInt:   pctInt,
			Near:         pctInt >= 90 && pctInt < 100,
			Full:         pctInt >= 100,
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].PercentInt != rows[j].PercentInt {
			return rows[i].PercentInt > rows[j].PercentInt
		}
		if rows[i].Allocated != rows[j].Allocated {
			return rows[i].Allocated > rows[j].Allocated
		}
		return rows[i].Label < rows[j].Label
	})

	constrained := 0
	for _, r := range rows {
		if r.PercentInt >= 80 {
			constrained++
		}
	}

	top := rows
	if len(top) > 3 {
		top = top[:3]
	}

	return PlatformHealthSummary{
		TotalGovernedTypes: len(rows),
		ConstrainedTypes:   constrained,
		TopThree:           top,
	}
}
