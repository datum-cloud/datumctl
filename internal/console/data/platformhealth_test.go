package data

import "testing"

func TestComputePlatformHealthSummary_Empty(t *testing.T) {
	t.Parallel()
	got := ComputePlatformHealthSummary(nil, "Project", "my-proj", nil)
	if got.TotalGovernedTypes != 0 || got.ConstrainedTypes != 0 || got.TopThree != nil {
		t.Errorf("empty input: want zero-value summary, got %+v", got)
	}
}

func TestComputePlatformHealthSummary_NoActiveKind(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "x/a", ConsumerKind: "Project", ConsumerName: "p", Allocated: 50, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "", "", nil)
	if got.TotalGovernedTypes != 0 {
		t.Errorf("empty active kind: want zero summary, got %+v", got)
	}
}

func TestComputePlatformHealthSummary_ThreeBucketsSorted(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "networking.datumapis.com/httpproxies", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 95, Limit: 100},
		{ResourceType: "resourcemanager.datumapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 40, Limit: 100},
		{ResourceType: "compute.datumapis.com/workloads", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 80, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "my-proj", nil)
	if got.TotalGovernedTypes != 3 {
		t.Errorf("total governed types = %d, want 3", got.TotalGovernedTypes)
	}
	if got.ConstrainedTypes != 2 {
		t.Errorf("constrained types = %d, want 2", got.ConstrainedTypes)
	}
	if len(got.TopThree) != 3 {
		t.Fatalf("len(TopThree) = %d, want 3", len(got.TopThree))
	}
	if got.TopThree[0].PercentInt != 95 || got.TopThree[0].Label != "httpproxies" {
		t.Errorf("top[0] = %+v, want 95%% httpproxies", got.TopThree[0])
	}
	if !got.TopThree[0].Near {
		t.Errorf("top[0].Near = false, want true (95%%)")
	}
	if got.TopThree[1].PercentInt != 80 || got.TopThree[1].Label != "workloads" {
		t.Errorf("top[1] = %+v, want 80%% workloads", got.TopThree[1])
	}
	if got.TopThree[1].Near {
		t.Errorf("top[1].Near = true, want false (80%%)")
	}
	if got.TopThree[2].PercentInt != 40 {
		t.Errorf("top[2].PercentInt = %d, want 40", got.TopThree[2].PercentInt)
	}
}

func TestComputePlatformHealthSummary_RegistrationMiss_ShortNameFallback(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "networking.datumapis.com/httpproxies", ConsumerKind: "Project", ConsumerName: "p", Allocated: 10, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "p", []ResourceRegistration{})
	if len(got.TopThree) != 1 || got.TopThree[0].Label != "httpproxies" {
		t.Errorf("registration miss: want label 'httpproxies', got %+v", got.TopThree)
	}
}

func TestComputePlatformHealthSummary_RegistrationHit_UsesDescription(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "networking.datumapis.com/httpproxies", ConsumerKind: "Project", ConsumerName: "p", Allocated: 10, Limit: 100},
	}
	regs := []ResourceRegistration{
		{Group: "networking.datumapis.com", Name: "httpproxies", Description: "HTTP proxies"},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "p", regs)
	if len(got.TopThree) != 1 || got.TopThree[0].Label != "HTTP proxies" {
		t.Errorf("registration hit: want label 'HTTP proxies', got %+v", got.TopThree)
	}
}

func TestComputePlatformHealthSummary_ZeroLimitExcluded(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "x/a", ConsumerKind: "Project", ConsumerName: "p", Allocated: 100, Limit: 0},
		{ResourceType: "x/b", ConsumerKind: "Project", ConsumerName: "p", Allocated: 50, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "p", nil)
	if got.TotalGovernedTypes != 1 {
		t.Errorf("zero-limit: want TotalGovernedTypes 1, got %d", got.TotalGovernedTypes)
	}
	if got.ConstrainedTypes != 0 {
		t.Errorf("zero-limit: want ConstrainedTypes 0, got %d", got.ConstrainedTypes)
	}
	if len(got.TopThree) != 1 || got.TopThree[0].Label != "b" {
		t.Errorf("zero-limit: want TopThree [b 50%%], got %+v", got.TopThree)
	}
}

func TestComputePlatformHealthSummary_DifferentConsumerFiltered(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "x/a", ConsumerKind: "Organization", ConsumerName: "other-org", Allocated: 100, Limit: 100},
		{ResourceType: "x/b", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 50, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "my-proj", nil)
	if got.TotalGovernedTypes != 1 {
		t.Errorf("consumer filter: want 1 type, got %d", got.TotalGovernedTypes)
	}
	if len(got.TopThree) != 1 || got.TopThree[0].Label != "b" {
		t.Errorf("consumer filter: want only 'b' row, got %+v", got.TopThree)
	}
}

func TestComputePlatformHealthSummary_TieBreakByAllocatedThenLabel(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "x/beta", ConsumerKind: "Project", ConsumerName: "p", Allocated: 80, Limit: 100},
		{ResourceType: "x/alpha", ConsumerKind: "Project", ConsumerName: "p", Allocated: 80, Limit: 100},
		{ResourceType: "x/gamma", ConsumerKind: "Project", ConsumerName: "p", Allocated: 50, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "p", nil)
	if len(got.TopThree) != 3 {
		t.Fatalf("want 3 rows, got %d", len(got.TopThree))
	}
	// alpha and beta both at 80%, both same allocated — tertiary sort is alphabetical.
	if got.TopThree[0].Label != "alpha" {
		t.Errorf("tie-break[0] = %q, want 'alpha' (alphabetical)", got.TopThree[0].Label)
	}
	if got.TopThree[1].Label != "beta" {
		t.Errorf("tie-break[1] = %q, want 'beta'", got.TopThree[1].Label)
	}
}

func TestComputePlatformHealthSummary_CaseInsensitiveConsumerKind(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "x/a", ConsumerKind: "project", ConsumerName: "p", Allocated: 50, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "p", nil)
	if got.TotalGovernedTypes != 1 {
		t.Errorf("case-insensitive kind: want 1 type, got %d", got.TotalGovernedTypes)
	}
}

func TestComputePlatformHealthSummary_FullSuffixAt100(t *testing.T) {
	t.Parallel()
	buckets := []AllowanceBucket{
		{ResourceType: "x/a", ConsumerKind: "Project", ConsumerName: "p", Allocated: 100, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "p", nil)
	if len(got.TopThree) != 1 {
		t.Fatalf("want 1 row, got %d", len(got.TopThree))
	}
	if !got.TopThree[0].Full {
		t.Errorf("100%%: Full = false, want true")
	}
	if got.TopThree[0].Near {
		t.Errorf("100%%: Near = true, want false (Full supersedes)")
	}
}

func TestComputePlatformHealthSummary_GroupsByResourceTypeKeepsMaxUtil(t *testing.T) {
	t.Parallel()
	// Two buckets with same ResourceType (different grants) — the max-util one wins.
	buckets := []AllowanceBucket{
		{ResourceType: "x/a", ConsumerKind: "Project", ConsumerName: "p", Allocated: 10, Limit: 100},
		{ResourceType: "x/a", ConsumerKind: "Project", ConsumerName: "p", Allocated: 95, Limit: 100},
	}
	got := ComputePlatformHealthSummary(buckets, "Project", "p", nil)
	if got.TotalGovernedTypes != 1 {
		t.Errorf("want 1 type, got %d", got.TotalGovernedTypes)
	}
	if len(got.TopThree) != 1 || got.TopThree[0].PercentInt != 95 {
		t.Errorf("want top row at 95%%, got %+v", got.TopThree)
	}
}
