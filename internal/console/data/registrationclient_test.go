package data

import "testing"

// --- ResolveDescription ---

func TestResolveDescription_HitWithDescription(t *testing.T) {
	t.Parallel()
	regs := []ResourceRegistration{
		{Group: "compute.example.io", Name: "cpus", Description: "CPU cores"},
	}
	got := ResolveDescription(regs, "compute.example.io", "cpus")
	if got != "CPU cores" {
		t.Errorf("ResolveDescription hit = %q, want %q", got, "CPU cores")
	}
}

func TestResolveDescription_Miss_WrongGroup(t *testing.T) {
	t.Parallel()
	regs := []ResourceRegistration{
		{Group: "other.io", Name: "cpus", Description: "CPU cores"},
	}
	got := ResolveDescription(regs, "compute.example.io", "cpus")
	if got != "" {
		t.Errorf("ResolveDescription wrong group = %q, want empty", got)
	}
}

func TestResolveDescription_Miss_WrongName(t *testing.T) {
	t.Parallel()
	regs := []ResourceRegistration{
		{Group: "compute.example.io", Name: "memory", Description: "RAM"},
	}
	got := ResolveDescription(regs, "compute.example.io", "cpus")
	if got != "" {
		t.Errorf("ResolveDescription wrong name = %q, want empty", got)
	}
}

func TestResolveDescription_Miss_EmptyDescription(t *testing.T) {
	t.Parallel()
	regs := []ResourceRegistration{
		{Group: "compute.example.io", Name: "cpus", Description: ""},
	}
	got := ResolveDescription(regs, "compute.example.io", "cpus")
	if got != "" {
		t.Errorf("ResolveDescription empty description = %q, want empty (not a hit)", got)
	}
}

func TestResolveDescription_NilSlice(t *testing.T) {
	t.Parallel()
	got := ResolveDescription(nil, "compute.example.io", "cpus")
	if got != "" {
		t.Errorf("ResolveDescription nil slice = %q, want empty", got)
	}
}

func TestResolveDescription_EmptySlice(t *testing.T) {
	t.Parallel()
	got := ResolveDescription([]ResourceRegistration{}, "compute.example.io", "cpus")
	if got != "" {
		t.Errorf("ResolveDescription empty slice = %q, want empty", got)
	}
}

func TestResolveDescription_FirstMatchWins(t *testing.T) {
	t.Parallel()
	regs := []ResourceRegistration{
		{Group: "g", Name: "r", Description: "First"},
		{Group: "g", Name: "r", Description: "Second"},
	}
	got := ResolveDescription(regs, "g", "r")
	if got != "First" {
		t.Errorf("ResolveDescription first-match = %q, want %q", got, "First")
	}
}

// --- SplitResourceType ---

func TestSplitResourceType_WithSlash(t *testing.T) {
	t.Parallel()
	group, name := SplitResourceType("compute.example.io/cpus")
	if group != "compute.example.io" {
		t.Errorf("SplitResourceType group = %q, want %q", group, "compute.example.io")
	}
	if name != "cpus" {
		t.Errorf("SplitResourceType name = %q, want %q", name, "cpus")
	}
}

func TestSplitResourceType_NoSlash(t *testing.T) {
	t.Parallel()
	group, name := SplitResourceType("pods")
	if group != "" {
		t.Errorf("SplitResourceType no-slash group = %q, want empty", group)
	}
	if name != "pods" {
		t.Errorf("SplitResourceType no-slash name = %q, want %q", name, "pods")
	}
}

func TestSplitResourceType_MultipleSlashes_UsesLast(t *testing.T) {
	t.Parallel()
	// LastIndex is used — group captures everything before the last slash.
	group, name := SplitResourceType("a.b.c/d/resource")
	if group != "a.b.c/d" {
		t.Errorf("SplitResourceType multi-slash group = %q, want %q", group, "a.b.c/d")
	}
	if name != "resource" {
		t.Errorf("SplitResourceType multi-slash name = %q, want %q", name, "resource")
	}
}
