package data

import "testing"

func TestFindBucketsForResource(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "b-compute-1", ResourceType: "compute.example.io/cpus"},
		{Name: "b-compute-2", ResourceType: "compute.example.io/cpus"},
		{Name: "b-storage", ResourceType: "storage.example.io/bytes"},
		{Name: "b-core", ResourceType: "pods"},
	}

	tests := []struct {
		name      string
		group     string
		resource  string
		wantNames []string
	}{
		{
			name: "non-core group match returns all matching buckets",
			group: "compute.example.io", resource: "cpus",
			wantNames: []string{"b-compute-1", "b-compute-2"},
		},
		{
			name: "empty group matches core resource",
			group: "", resource: "pods",
			wantNames: []string{"b-core"},
		},
		{
			name: "single non-core match",
			group: "storage.example.io", resource: "bytes",
			wantNames: []string{"b-storage"},
		},
		{
			name:      "no match returns empty",
			group:     "example.io", resource: "widgets",
			wantNames: nil,
		},
		{
			name:      "wrong group returns empty",
			group:     "other.io", resource: "cpus",
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FindBucketsForResource(buckets, tt.group, tt.resource)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("FindBucketsForResource(%q, %q) len = %d, want %d; got names: %v",
					tt.group, tt.resource, len(got), len(tt.wantNames), bucketNames(got))
			}
			for i, b := range got {
				if b.Name != tt.wantNames[i] {
					t.Errorf("got[%d].Name = %q, want %q", i, b.Name, tt.wantNames[i])
				}
			}
		})
	}
}

func bucketNames(bs []AllowanceBucket) []string {
	names := make([]string, len(bs))
	for i, b := range bs {
		names[i] = b.Name
	}
	return names
}

// --- ClassifyTreeBuckets ---

func TestClassifyTreeBuckets_BothPresent_HasTree(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org"},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj"},
	}
	got := ClassifyTreeBuckets(buckets, "Project", "my-proj")
	if !got.HasTree {
		t.Error("HasTree = false, want true when org+project both present")
	}
	if got.Parent == nil || got.Parent.Name != "org" {
		t.Errorf("Parent = %v, want 'org'", got.Parent)
	}
	if got.ActiveChild == nil || got.ActiveChild.Name != "proj" {
		t.Errorf("ActiveChild = %v, want 'proj'", got.ActiveChild)
	}
}

func TestClassifyTreeBuckets_NoParent_HasTreeFalse(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj"},
	}
	got := ClassifyTreeBuckets(buckets, "Project", "my-proj")
	if got.HasTree {
		t.Error("HasTree = true, want false when no org parent")
	}
}

func TestClassifyTreeBuckets_NoChild_HasTreeFalse(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org"},
	}
	got := ClassifyTreeBuckets(buckets, "Project", "my-proj")
	if got.HasTree {
		t.Error("HasTree = true, want false when no matching active child")
	}
}

func TestClassifyTreeBuckets_Empty_HasTreeFalse(t *testing.T) {
	t.Parallel()
	got := ClassifyTreeBuckets(nil, "Project", "my-proj")
	if got.HasTree {
		t.Error("HasTree = true for nil input, want false")
	}
}

func TestClassifyTreeBuckets_CaseInsensitiveOrg(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "organization", ConsumerName: "my-org"},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "p"},
	}
	got := ClassifyTreeBuckets(buckets, "Project", "p")
	if !got.HasTree {
		t.Error("HasTree = false, want true — org kind is case-insensitive")
	}
}

func TestClassifyTreeBuckets_EmptyActiveConsumerKind_NoChild(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org"},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "p"},
	}
	got := ClassifyTreeBuckets(buckets, "", "p")
	if got.HasTree {
		t.Error("HasTree = true, want false when activeConsumerKind is empty")
	}
}

func TestClassifyTreeBuckets_SiblingsPopulated(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org"},
		{Name: "proj-active", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "active"},
		{Name: "proj-sib", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "sibling"},
	}
	got := ClassifyTreeBuckets(buckets, "Project", "active")
	if !got.HasTree {
		t.Fatal("HasTree = false, expected tree")
	}
	if len(got.Siblings) != 1 {
		t.Errorf("Siblings len = %d, want 1", len(got.Siblings))
	}
	if got.Siblings[0].Name != "proj-sib" {
		t.Errorf("Siblings[0].Name = %q, want %q", got.Siblings[0].Name, "proj-sib")
	}
}

// --- FindSiblingBuckets ---

func TestFindSiblingBuckets_ZeroParent_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "p"},
	}
	// Parent with empty ResourceType
	got := FindSiblingBuckets(buckets, AllowanceBucket{}, "Project", "p")
	if got == nil {
		t.Error("FindSiblingBuckets zero parent returned nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("FindSiblingBuckets zero parent len = %d, want 0", len(got))
	}
}

func TestFindSiblingBuckets_NormalSiblings(t *testing.T) {
	t.Parallel()
	parent := AllowanceBucket{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org"}
	buckets := []AllowanceBucket{
		parent,
		{Name: "active", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "active-proj"},
		{Name: "sib1", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "other-proj"},
		{Name: "sib2", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "third-proj"},
	}
	got := FindSiblingBuckets(buckets, parent, "Project", "active-proj")
	if len(got) != 2 {
		t.Errorf("FindSiblingBuckets len = %d, want 2; names: %v", len(got), bucketNames(got))
	}
}

func TestFindSiblingBuckets_ExcludesActiveConsumer(t *testing.T) {
	t.Parallel()
	parent := AllowanceBucket{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization"}
	buckets := []AllowanceBucket{
		parent,
		{Name: "active", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "active"},
	}
	got := FindSiblingBuckets(buckets, parent, "Project", "active")
	if len(got) != 0 {
		t.Errorf("FindSiblingBuckets: active consumer should be excluded, got %v", bucketNames(got))
	}
}

func TestFindSiblingBuckets_ExcludesParentKind(t *testing.T) {
	t.Parallel()
	parent := AllowanceBucket{Name: "org1", ResourceType: "a/r", ConsumerKind: "Organization"}
	buckets := []AllowanceBucket{
		parent,
		{Name: "org2", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "other-org"},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "p"},
	}
	got := FindSiblingBuckets(buckets, parent, "Project", "active")
	// org2 excluded (same kind as parent), proj returned
	for _, b := range got {
		if b.Name == "org2" {
			t.Error("FindSiblingBuckets: org2 (parent kind) must be excluded")
		}
	}
}

func TestFindSiblingBuckets_ExcludesDifferentResourceType(t *testing.T) {
	t.Parallel()
	parent := AllowanceBucket{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization"}
	buckets := []AllowanceBucket{
		parent,
		{Name: "other-rt", ResourceType: "b/s", ConsumerKind: "Project", ConsumerName: "p"},
	}
	got := FindSiblingBuckets(buckets, parent, "Project", "active")
	if len(got) != 0 {
		t.Errorf("FindSiblingBuckets: different ResourceType must be excluded, got %v", bucketNames(got))
	}
}
