package function

import "testing"

func TestParseMachineFamily(t *testing.T) {
	tests := []struct {
		input        string
		wantFamily   string
		wantCategory string
	}{
		// Standard cases
		{"n2d-standard-4", "n2d", "standard"},
		{"n2-highcpu-16", "n2", "highcpu"},
		{"c3-standard-88-lssd", "c3", "standard"},
		{"c3-standard-176-metal", "c3", "standard"},
		// Shared-core
		{"e2-micro", "e2", "micro"},
		{"f1-micro", "f1", "micro"},
		{"g1-small", "g1", "small"},
		// Memory-optimized
		{"m1-megamem-96", "m1", "megamem"},
		{"m2-ultramem-208", "m2", "ultramem"},
		{"m4-hypermem-112", "m4", "hypermem"},
		// GPU
		{"a2-highgpu-8g", "a2", "highgpu"},
		// Custom types
		{"n2-custom-8-32768", "n2", "custom"},
		{"n2-custom-8-32768-ext", "n2", "custom"},
		{"custom-6-23040", "n1", "custom"},
		// X4 unusual format (family is x4, "category" is the vcpu count string)
		{"x4-480-8t-metal", "x4", "480"},
		// Bare series
		{"n2d", "n2d", ""},
	}
	for _, tt := range tests {
		family, category := parseMachineFamily(tt.input)
		if family != tt.wantFamily || category != tt.wantCategory {
			t.Errorf("parseMachineFamily(%q) = (%q, %q), want (%q, %q)",
				tt.input, family, category, tt.wantFamily, tt.wantCategory)
		}
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		input   string
		wantMin int
		wantMax int
	}{
		{"", 0, 0},
		{"4", 4, 4},
		{"2+8", 2, 8},
		{"16+32", 16, 32},
	}
	for _, tt := range tests {
		min, max := parseRange(tt.input)
		if min != tt.wantMin || max != tt.wantMax {
			t.Errorf("parseRange(%q) = (%d, %d), want (%d, %d)",
				tt.input, min, max, tt.wantMin, tt.wantMax)
		}
	}
}

func TestMatchesFamily(t *testing.T) {
	mt := &MachineTypeInfo{Family: "n2d"}
	if !matchesFamily(mt, []string{"n2", "n2d"}) {
		t.Error("expected n2d to match [n2, n2d]")
	}
	if matchesFamily(mt, []string{"c3", "n2"}) {
		t.Error("expected n2d not to match [c3, n2]")
	}
}

func TestIsSharedCoreCategory(t *testing.T) {
	for _, cat := range []string{"micro", "small", "medium"} {
		if !isSharedCoreCategory(cat) {
			t.Errorf("expected %q to be shared-core", cat)
		}
	}
	for _, cat := range []string{"standard", "highcpu", "highmem", "custom", "megamem"} {
		if isSharedCoreCategory(cat) {
			t.Errorf("expected %q to NOT be shared-core", cat)
		}
	}
}

func TestIsGPUCategory(t *testing.T) {
	for _, cat := range []string{"highgpu", "megagpu", "ultragpu", "edgegpu", "maxgpu"} {
		if !isGPUCategory(cat) {
			t.Errorf("expected %q to be GPU category", cat)
		}
	}
	for _, cat := range []string{"standard", "highcpu", "highmem"} {
		if isGPUCategory(cat) {
			t.Errorf("expected %q to NOT be GPU category", cat)
		}
	}
}

// TestResolveMachineType_ExactMode tests that exact mode returns machine as-is.
func TestResolveMachineType_ExactMode(t *testing.T) {
	labels := &RunnerLabels{
		Machine:     "n2d-standard-4",
		MachineMode: "exact",
	}
	resolved, err := ResolveMachineType(nil, "", "", labels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "n2d-standard-4" {
		t.Errorf("resolved = %q, want %q", resolved, "n2d-standard-4")
	}
}

// TestFilterLogic_SkipsSharedCore verifies shared-core types are excluded from resolution.
func TestFilterLogic_SkipsSharedCore(t *testing.T) {
	types := []*MachineTypeInfo{
		{Name: "e2-micro", Family: "e2", Category: "micro", VCPUs: 2, MemoryMB: 1024},
		{Name: "e2-small", Family: "e2", Category: "small", VCPUs: 2, MemoryMB: 2048},
		{Name: "e2-standard-2", Family: "e2", Category: "standard", VCPUs: 2, MemoryMB: 8192},
	}

	families := []string{"e2"}
	var candidates []*MachineTypeInfo
	for _, mt := range types {
		if !matchesFamily(mt, families) {
			continue
		}
		if isSharedCoreCategory(mt.Category) {
			continue
		}
		candidates = append(candidates, mt)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Name != "e2-standard-2" {
		t.Errorf("candidate = %q, want %q", candidates[0].Name, "e2-standard-2")
	}
}

// TestFilterLogic_SkipsGPU verifies GPU types are excluded from resolution.
func TestFilterLogic_SkipsGPU(t *testing.T) {
	types := []*MachineTypeInfo{
		{Name: "a2-highgpu-1g", Family: "a2", Category: "highgpu", VCPUs: 12, MemoryMB: 87040},
		{Name: "n2-standard-4", Family: "n2", Category: "standard", VCPUs: 4, MemoryMB: 16384},
	}

	var nonGPU []*MachineTypeInfo
	for _, mt := range types {
		if !isGPUCategory(mt.Category) {
			nonGPU = append(nonGPU, mt)
		}
	}

	if len(nonGPU) != 1 || nonGPU[0].Name != "n2-standard-4" {
		t.Errorf("expected only n2-standard-4, got %v", nonGPU)
	}
}

// TestFilterLogic_IncludesMemoryOptimizedCategories ensures megamem/ultramem/hypermem pass filters.
func TestFilterLogic_IncludesMemoryOptimizedCategories(t *testing.T) {
	types := []*MachineTypeInfo{
		{Name: "m1-megamem-96", Family: "m1", Category: "megamem", VCPUs: 96, MemoryMB: 1433600},
		{Name: "m2-ultramem-208", Family: "m2", Category: "ultramem", VCPUs: 208, MemoryMB: 5888000},
		{Name: "m4-hypermem-112", Family: "m4", Category: "hypermem", VCPUs: 112, MemoryMB: 1835008},
	}

	for _, mt := range types {
		if isSharedCoreCategory(mt.Category) {
			t.Errorf("%q should NOT be classified as shared-core", mt.Name)
		}
		if isGPUCategory(mt.Category) {
			t.Errorf("%q should NOT be classified as GPU", mt.Name)
		}
	}
}

func TestFilterMachineTypes_CPUConstraint(t *testing.T) {
	types := []*MachineTypeInfo{
		{Name: "n2d-standard-2", Family: "n2d", Category: "standard", VCPUs: 2, MemoryMB: 8192},
		{Name: "n2d-standard-4", Family: "n2d", Category: "standard", VCPUs: 4, MemoryMB: 16384},
		{Name: "n2d-standard-8", Family: "n2d", Category: "standard", VCPUs: 8, MemoryMB: 32768},
		{Name: "n2d-highcpu-4", Family: "n2d", Category: "highcpu", VCPUs: 4, MemoryMB: 4096},
	}

	families := []string{"n2d"}
	minCPU, maxCPU := 4, 4

	var best *MachineTypeInfo
	for _, mt := range types {
		if !matchesFamily(mt, families) || isSharedCoreCategory(mt.Category) || isGPUCategory(mt.Category) {
			continue
		}
		if int(mt.VCPUs) < minCPU || int(mt.VCPUs) > maxCPU {
			continue
		}
		if best == nil || mt.VCPUs < best.VCPUs || (mt.VCPUs == best.VCPUs && mt.MemoryMB < best.MemoryMB) {
			best = mt
		}
	}

	if best == nil {
		t.Fatal("expected a match, got nil")
	}
	// Should pick n2d-highcpu-4 as smallest (4 vCPUs, less RAM than standard-4)
	if best.Name != "n2d-highcpu-4" {
		t.Errorf("best = %q, want %q", best.Name, "n2d-highcpu-4")
	}
}

func TestFilterMachineTypes_WithRAMConstraint(t *testing.T) {
	types := []*MachineTypeInfo{
		{Name: "n2d-standard-2", Family: "n2d", Category: "standard", VCPUs: 2, MemoryMB: 8192},
		{Name: "n2d-standard-4", Family: "n2d", Category: "standard", VCPUs: 4, MemoryMB: 16384},
		{Name: "n2d-highcpu-4", Family: "n2d", Category: "highcpu", VCPUs: 4, MemoryMB: 4096},
	}

	families := []string{"n2d"}
	minCPU, maxCPU := 4, 4
	minRAM, maxRAM := 16, 16

	var best *MachineTypeInfo
	for _, mt := range types {
		if !matchesFamily(mt, families) || isSharedCoreCategory(mt.Category) || isGPUCategory(mt.Category) {
			continue
		}
		if int(mt.VCPUs) < minCPU || int(mt.VCPUs) > maxCPU {
			continue
		}
		ramGB := int(mt.MemoryMB) / 1024
		if ramGB < minRAM || ramGB > maxRAM {
			continue
		}
		if best == nil || mt.VCPUs < best.VCPUs || (mt.VCPUs == best.VCPUs && mt.MemoryMB < best.MemoryMB) {
			best = mt
		}
	}

	if best == nil {
		t.Fatal("expected a match, got nil")
	}
	if best.Name != "n2d-standard-4" {
		t.Errorf("best = %q, want %q", best.Name, "n2d-standard-4")
	}
}

func TestFilterMachineTypes_MultiFamily(t *testing.T) {
	types := []*MachineTypeInfo{
		{Name: "n2d-standard-4", Family: "n2d", Category: "standard", VCPUs: 4, MemoryMB: 16384},
		{Name: "c3-standard-4", Family: "c3", Category: "standard", VCPUs: 4, MemoryMB: 16384},
		{Name: "n2-standard-4", Family: "n2", Category: "standard", VCPUs: 4, MemoryMB: 16384},
	}

	families := []string{"n2", "c3"}

	var matches []string
	for _, mt := range types {
		if matchesFamily(mt, families) {
			matches = append(matches, mt.Name)
		}
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d: %v", len(matches), matches)
	}
}
