package function

import "testing"

func TestParseLabels_ExactMachine(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/machine=n2-standard-4"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.Machine != "n2-standard-4" {
		t.Errorf("Machine = %q, want %q", labels.Machine, "n2-standard-4")
	}
	if labels.MachineMode != "exact" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "exact")
	}
}

func TestParseLabels_ExactMachineSharedCore(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/machine=e2-micro"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "exact" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "exact")
	}
	if labels.Machine != "e2-micro" {
		t.Errorf("Machine = %q, want %q", labels.Machine, "e2-micro")
	}
}

func TestParseLabels_ExactMachineLssd(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/machine=c3-standard-88-lssd"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "exact" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "exact")
	}
}

func TestParseLabels_FamilyMode(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/family=n2"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.Family != "n2" {
		t.Errorf("Family = %q, want %q", labels.Family, "n2")
	}
	if labels.MachineMode != "family" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "family")
	}
}

func TestParseLabels_FamilyWithCPU(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/family=n2/cpu=4"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "family" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "family")
	}
	if labels.Family != "n2" {
		t.Errorf("Family = %q, want %q", labels.Family, "n2")
	}
	if labels.CPU != "4" {
		t.Errorf("CPU = %q, want %q", labels.CPU, "4")
	}
}

func TestParseLabels_FamilyWithCPUAndRAM(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/family=c3/cpu=8/ram=32"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "family" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "family")
	}
	if labels.Family != "c3" {
		t.Errorf("Family = %q, want %q", labels.Family, "c3")
	}
	if labels.CPU != "8" {
		t.Errorf("CPU = %q, want %q", labels.CPU, "8")
	}
	if labels.RAM != "32" {
		t.Errorf("RAM = %q, want %q", labels.RAM, "32")
	}
}

func TestParseLabels_MultiFamilyWithCPU(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/family=n2d+c3/cpu=4"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "family" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "family")
	}
	if labels.Family != "n2d+c3" {
		t.Errorf("Family = %q, want %q", labels.Family, "n2d+c3")
	}
}

func TestParseLabels_AutoMode_CPUOnly(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/cpu=4"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "auto" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "auto")
	}
	if labels.CPU != "4" {
		t.Errorf("CPU = %q, want %q", labels.CPU, "4")
	}
}

func TestParseLabels_AutoMode_CPURange(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/cpu=2+8"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "auto" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "auto")
	}
	if labels.CPU != "2+8" {
		t.Errorf("CPU = %q, want %q", labels.CPU, "2+8")
	}
}

func TestParseLabels_AutoMode_CPUAndRAM(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/cpu=4/ram=16"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "auto" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "auto")
	}
}

func TestParseLabels_FamilyTakesPrecedenceOverDefaultMachine(t *testing.T) {
	// family= set with cpu= → family mode, even though machine is default
	labels := parseLabels([]string{"gcrunner=test/family=e2/cpu=2"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "family" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "family")
	}
	if labels.Family != "e2" {
		t.Errorf("Family = %q, want %q", labels.Family, "e2")
	}
}

func TestParseLabels_MachineTakesPrecedenceOverCPURAM(t *testing.T) {
	// If machine= is explicitly set, cpu/ram are ignored (exact mode)
	labels := parseLabels([]string{"gcrunner=test/machine=c3-standard-88-lssd/cpu=4"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "exact" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "exact")
	}
	if labels.Machine != "c3-standard-88-lssd" {
		t.Errorf("Machine = %q, want %q", labels.Machine, "c3-standard-88-lssd")
	}
}

func TestParseLabels_ZoneLabel(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test/zone=us-central1-a+us-central1-b"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.Zone != "us-central1-a+us-central1-b" {
		t.Errorf("Zone = %q, want %q", labels.Zone, "us-central1-a+us-central1-b")
	}
}

func TestParseLabels_DefaultsAreExactMode(t *testing.T) {
	labels := parseLabels([]string{"gcrunner=test"})
	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels.MachineMode != "exact" {
		t.Errorf("MachineMode = %q, want %q", labels.MachineMode, "exact")
	}
	if labels.Machine != "n2d-standard-2" {
		t.Errorf("Machine = %q, want %q", labels.Machine, "n2d-standard-2")
	}
}

func TestParseLabels_NotGcrunner(t *testing.T) {
	labels := parseLabels([]string{"self-hosted", "linux"})
	if labels != nil {
		t.Errorf("expected nil for non-gcrunner labels, got %+v", labels)
	}
}
