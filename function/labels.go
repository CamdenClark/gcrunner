package function

import "strings"

// RunnerLabels holds the parsed gcrunner label configuration.
type RunnerLabels struct {
	RunID   string
	Machine string
	Spot    bool
	Disk    string
	Image   string
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
			Machine: "e2-medium",
			Spot:    true,
			Disk:    "50gb",
			Image:   "ubuntu24-full-x64",
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
			case "spot":
				result.Spot = kv[1] != "false"
			case "disk":
				result.Disk = kv[1]
			case "image":
				result.Image = kv[1]
			}
		}

		return result
	}
	return nil
}
