---
title: "Job Labels"
weight: 3
---

# Job Labels

All runner configuration lives in the `runs-on` field of your workflow using `/`-separated `key=value` labels. The only required label is `gcrunner=`, which also identifies the job to gcrunner.

```yaml
runs-on: gcrunner=${{ github.run_id }}/machine=n2d-standard-8/disk=100gb/spot=false
```

## Reference

| Label | Default | Description |
|---|---|---|
| `gcrunner` | *(required)* | Identifies the job. Use `${{ github.run_id }}` as the value. |
| `machine` | `n2d-standard-2` | Exact GCE machine type |
| `family` | — | Machine family for automatic type resolution (see below) |
| `cpu` | — | vCPU constraint for auto/family resolution, e.g. `4` or `2+8` |
| `ram` | — | RAM constraint in GB for auto/family resolution, e.g. `16` or `8+32` |
| `spot` | `true` | Use Spot VMs — 60–91% cheaper, with automatic fallback to on-demand |
| `disk` | `75gb` | Boot disk size |
| `disk-type` | `pd-ssd` | Boot disk type (e.g. `pd-ssd`, `pd-balanced`, `pd-standard`) |
| `image` | `ubuntu24-full-x64` | Runner image (see [Images](#images)) |
| `zone` | *(all zones in region)* | Zone or zone range to restrict placement (see [Zones](#zones)) |

## Machine Type Resolution

There are three ways to specify the machine type for a job.

### Exact mode

Set `machine` to a specific GCE machine type. The machine is used as-is, no resolution occurs.

```yaml
runs-on: gcrunner=${{ github.run_id }}/machine=c3-standard-8
```

### Family mode

Set `family` to one or more machine families (separated by `+`). gcrunner queries available machine types in the target zone and picks the smallest one that satisfies any `cpu` and `ram` constraints. Shared-core and GPU types are excluded.

```yaml
# Smallest n2d machine with at least 8 vCPUs
runs-on: gcrunner=${{ github.run_id }}/family=n2d/cpu=8

# Any n2d or c3 machine with 8–32 vCPUs and 32–128 GB RAM
runs-on: gcrunner=${{ github.run_id }}/family=n2d+c3/cpu=8+32/ram=32+128
```

### Auto mode

Set `cpu` and/or `ram` without specifying `family` or `machine`. gcrunner uses the `n2d` family as the default and resolves the smallest matching type.

```yaml
# Smallest n2d machine with at least 4 vCPUs
runs-on: gcrunner=${{ github.run_id }}/cpu=4

# Smallest n2d machine with at least 16 GB RAM
runs-on: gcrunner=${{ github.run_id }}/ram=16
```

### CPU and RAM constraints

Both `cpu` and `ram` accept either an exact value or a min+max range:

| Format | Meaning |
|---|---|
| `cpu=4` | Exactly 4 vCPUs |
| `cpu=4+16` | Between 4 and 16 vCPUs |
| `ram=16` | Exactly 16 GB RAM |
| `ram=8+64` | Between 8 and 64 GB RAM |

When both are specified, the machine must satisfy both constraints. gcrunner picks the smallest qualifying machine (fewest vCPUs, then least RAM).

## Spot VMs

Spot VMs are enabled by default (`spot=true`). They are significantly cheaper than on-demand but can be preempted with 30 seconds of warning. gcrunner automatically falls back to an on-demand VM if Spot capacity is unavailable in the target zone.

For jobs that cannot tolerate preemption (e.g. production deploys), disable Spot:

```yaml
runs-on: gcrunner=${{ github.run_id }}/spot=false
```

## Zones

By default, gcrunner tries all zones in your configured region. You can restrict placement to a specific zone or a set of zones using the `zone` label:

```yaml
# Single zone
runs-on: gcrunner=${{ github.run_id }}/zone=us-central1-a

# Restrict to two zones
runs-on: gcrunner=${{ github.run_id }}/zone=us-central1-a+us-central1-b
```

Zones are tried in the order specified. If a zone has no capacity (or the machine type is unavailable there), gcrunner moves to the next one.

## Images

| Value | Description |
|---|---|
| `ubuntu24-full-x64` | Ubuntu 24.04 LTS, x86-64 *(default)* |
| `ubuntu22-full-x64` | Ubuntu 22.04 LTS, x86-64 |

You can also provide a fully-qualified GCE image path:

```yaml
runs-on: gcrunner=${{ github.run_id }}/image=projects/my-project/global/images/family/my-runner
```

## Examples

```yaml
# Default — small n2d Spot VM, Ubuntu 24.04
runs-on: gcrunner=${{ github.run_id }}

# Larger machine for a compute-heavy build
runs-on: gcrunner=${{ github.run_id }}/machine=n2d-standard-8

# Production deploy — no Spot, specific zone
runs-on: gcrunner=${{ github.run_id }}/spot=false/zone=us-central1-a

# Auto-resolve: at least 8 vCPUs, at least 32 GB RAM
runs-on: gcrunner=${{ github.run_id }}/cpu=8/ram=32

# Large disk for Docker-heavy workloads
runs-on: gcrunner=${{ github.run_id }}/disk=200gb

# Custom image from your own project
runs-on: gcrunner=${{ github.run_id }}/image=projects/my-project/global/images/family/my-runner
```
