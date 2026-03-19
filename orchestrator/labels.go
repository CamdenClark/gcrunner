package function

import "strings"

// RunnerLabels holds the parsed gcrunner label configuration.
type RunnerLabels struct {
	RunID    string
	Machine  string   // Exact machine type (e.g. "n2d-standard-4", "e2-micro")
	Family   string   // Machine family for resolution (e.g. "n2d", "n2d+c3")
	Spot     bool
	Disk     string
	DiskType string
	Image    string
	CPU      string // "4" or "2+8" (range)
	RAM      string // "16" or "8+32" (range)
	Zone     string // "us-central1-a" or "us-central1-a+us-central1-b"
	// MachineMode is computed after parsing:
	//   "exact"  — Machine is set, use as-is
	//   "family" — Family is set, resolve with cpu/ram constraints
	//   "auto"   — Only cpu/ram set, resolve using default family (n2d)
	MachineMode string
}

// parseLabels extracts gcrunner config from workflow_job labels.
// Returns nil if this is not a gcrunner job.
func parseLabels(labels []string) *RunnerLabels {
	for _, label := range labels {
		if !strings.HasPrefix(label, "gcrunner=") {
			continue
		}

		result := &RunnerLabels{
			// Defaults from spec
			Machine:  "n2d-standard-2",
			Spot:     true,
			Disk:     "75gb",
			DiskType: "pd-ssd",
			Image:    "ubuntu24-full-x64",
		}

		parts := strings.Split(label, "/")
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}
			switch kv[0] {
			case "gcrunner":
				result.RunID = kv[1]
			case "machine":
				result.Machine = kv[1]
			case "family":
				result.Family = kv[1]
			case "spot":
				result.Spot = kv[1] != "false"
			case "disk":
				result.Disk = kv[1]
			case "disk-type":
				result.DiskType = kv[1]
			case "image":
				result.Image = kv[1]
			case "cpu":
				result.CPU = kv[1]
			case "ram":
				result.RAM = kv[1]
			case "zone":
				result.Zone = kv[1]
			}
		}

		result.MachineMode = classifyMachineMode(result)

		return result
	}
	return nil
}

// classifyMachineMode determines how the machine type should be resolved.
//
// Priority:
//  1. family= is set → "family" mode (resolve using family + cpu/ram)
//  2. machine= was explicitly changed from default → "exact" mode
//  3. cpu= or ram= set (no family, default machine) → "auto" mode (default family n2d)
//  4. Nothing set → "exact" mode with default machine
func classifyMachineMode(labels *RunnerLabels) string {
	if labels.Family != "" {
		return "family"
	}
	if labels.Machine != "n2d-standard-2" {
		// Machine was explicitly set — use it as-is
		return "exact"
	}
	if labels.CPU != "" || labels.RAM != "" {
		return "auto"
	}
	return "exact"
}
