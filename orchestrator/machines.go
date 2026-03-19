package function

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"
)

// MachineTypeInfo holds parsed information about a GCE machine type.
type MachineTypeInfo struct {
	Name     string // e.g. "n2d-standard-4"
	Family   string // e.g. "n2d"
	Category string // e.g. "standard"
	VCPUs    int32
	MemoryMB int32
}

// MachineTypeCache caches per-zone machine type lists with a TTL.
type MachineTypeCache struct {
	mu      sync.RWMutex
	types   map[string]machineTypeCacheEntry
	ttl     time.Duration
	nowFunc func() time.Time
}

type machineTypeCacheEntry struct {
	types     []*MachineTypeInfo
	fetchedAt time.Time
}

var machineTypeCache = &MachineTypeCache{
	types:   make(map[string]machineTypeCacheEntry),
	ttl:     1 * time.Hour,
	nowFunc: time.Now,
}

// ListMachineTypes returns machine types available in a zone, using a cache.
func ListMachineTypes(ctx context.Context, project, zone string) ([]*MachineTypeInfo, error) {
	return machineTypeCache.list(ctx, project, zone)
}

func (c *MachineTypeCache) list(ctx context.Context, project, zone string) ([]*MachineTypeInfo, error) {
	now := c.nowFunc()

	c.mu.RLock()
	entry, ok := c.types[zone]
	c.mu.RUnlock()

	if ok && now.Sub(entry.fetchedAt) < c.ttl {
		return entry.types, nil
	}

	types, err := fetchMachineTypes(ctx, project, zone)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.types[zone] = machineTypeCacheEntry{types: types, fetchedAt: now}
	c.mu.Unlock()

	return types, nil
}

func fetchMachineTypes(ctx context.Context, project, zone string) ([]*MachineTypeInfo, error) {
	client, err := compute.NewMachineTypesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create machine types client: %w", err)
	}
	defer client.Close()

	it := client.List(ctx, &computepb.ListMachineTypesRequest{
		Project: project,
		Zone:    zone,
	})

	var types []*MachineTypeInfo
	for {
		mt, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list machine types: %w", err)
		}
		family, category := parseMachineFamily(mt.GetName())
		types = append(types, &MachineTypeInfo{
			Name:     mt.GetName(),
			Family:   family,
			Category: category,
			VCPUs:    mt.GetGuestCpus(),
			MemoryMB: mt.GetMemoryMb(),
		})
	}

	return types, nil
}

// ResolveMachineType resolves a machine type string for a given zone based on labels.
// In exact mode, returns the machine as-is. In family or auto mode, queries available
// machine types and picks the smallest one satisfying the constraints.
func ResolveMachineType(ctx context.Context, project, zone string, labels *RunnerLabels) (string, error) {
	if labels.MachineMode == "exact" {
		return labels.Machine, nil
	}

	types, err := ListMachineTypes(ctx, project, zone)
	if err != nil {
		return "", fmt.Errorf("list machine types for %s: %w", zone, err)
	}

	family := labels.Family
	if labels.MachineMode == "auto" {
		family = "n2d"
	}
	families := strings.Split(family, "+")

	minCPU, maxCPU := parseRange(labels.CPU)
	minRAM, maxRAM := parseRange(labels.RAM)

	var best *MachineTypeInfo
	for _, mt := range types {
		if !matchesFamily(mt, families) {
			continue
		}
		// Skip shared-core types that don't have meaningful vCPU counts
		if isSharedCoreCategory(mt.Category) {
			continue
		}
		// Skip GPU-oriented categories — users wanting GPUs should use exact types
		if isGPUCategory(mt.Category) {
			continue
		}
		if minCPU > 0 && int(mt.VCPUs) < minCPU {
			continue
		}
		if maxCPU > 0 && int(mt.VCPUs) > maxCPU {
			continue
		}
		ramGB := int(mt.MemoryMB) / 1024
		if minRAM > 0 && ramGB < minRAM {
			continue
		}
		if maxRAM > 0 && ramGB > maxRAM {
			continue
		}
		if best == nil || mt.VCPUs < best.VCPUs || (mt.VCPUs == best.VCPUs && mt.MemoryMB < best.MemoryMB) {
			best = mt
		}
	}

	if best == nil {
		return "", fmt.Errorf("no machine type matching constraints (families=%v, cpu=%s, ram=%s) in zone %s",
			families, labels.CPU, labels.RAM, zone)
	}

	return best.Name, nil
}

// parseMachineFamily extracts the series/family from a machine type name.
// Examples:
//
//	"n2d-standard-4"       → "n2d", "standard"
//	"e2-micro"             → "e2", "micro"
//	"c3-standard-88-lssd"  → "c3", "standard"
//	"a2-highgpu-8g"        → "a2", "highgpu"
//	"custom-6-23040"       → "n1", "custom"    (legacy N1 custom format)
//	"n2d"                  → "n2d", ""
func parseMachineFamily(name string) (family, category string) {
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return name, ""
	}

	// Handle legacy N1 custom types: "custom-6-23040"
	if parts[0] == "custom" {
		return "n1", "custom"
	}

	family = parts[0]
	category = parts[1]
	return
}

func matchesFamily(mt *MachineTypeInfo, families []string) bool {
	for _, f := range families {
		if mt.Family == f {
			return true
		}
	}
	return false
}

// isSharedCoreCategory returns true for categories that represent shared-core
// machine types (e.g. e2-micro, f1-micro, g1-small) which have fractional vCPUs
// and shouldn't be matched by cpu/ram constraints.
func isSharedCoreCategory(category string) bool {
	switch category {
	case "micro", "small", "medium":
		return true
	}
	return false
}

// isGPUCategory returns true for accelerator-optimized categories.
// Users wanting GPU machines should specify exact machine types.
func isGPUCategory(category string) bool {
	switch category {
	case "highgpu", "megagpu", "ultragpu", "edgegpu", "maxgpu":
		return true
	}
	return false
}

// parseRange parses "4" into (4, 4) or "2+8" into (2, 8) or "" into (0, 0).
func parseRange(s string) (min, max int) {
	if s == "" {
		return 0, 0
	}
	if strings.Contains(s, "+") {
		parts := strings.SplitN(s, "+", 2)
		min, _ = strconv.Atoi(parts[0])
		max, _ = strconv.Atoi(parts[1])
		return
	}
	v, _ := strconv.Atoi(s)
	return v, v
}
