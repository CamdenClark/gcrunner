# Cloud NAT Setup for Runner VMs

## Problem

Every runner VM gets a public IP via `ONE_TO_ONE_NAT` (`orchestrator/vm.go:108-116`). Runner VMs only need outbound access (clone repos, pull packages, talk to GitHub API). A public IP exposes each VM to inbound attacks during the job execution window.

No VPC, subnet, or firewall configuration exists in Terraform — VMs use the default network.

## Changes Required

### Terraform: Add networking resources

```hcl
resource "google_compute_network" "gcrunner" {
  name                    = "gcrunner"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "runners" {
  name          = "gcrunner-runners"
  ip_cidr_range = "10.0.0.0/20"
  region        = var.region
  network       = google_compute_network.gcrunner.id

  private_ip_google_access = true  # Needed for GCP API access without public IP
}

resource "google_compute_router" "gcrunner" {
  name    = "gcrunner-router"
  region  = var.region
  network = google_compute_network.gcrunner.id
}

resource "google_compute_router_nat" "gcrunner" {
  name                               = "gcrunner-nat"
  router                             = google_compute_router.gcrunner.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

resource "google_compute_firewall" "deny_all_ingress" {
  name    = "gcrunner-deny-all-ingress"
  network = google_compute_network.gcrunner.id

  deny {
    protocol = "all"
  }

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]
  priority      = 1000
}

resource "google_compute_firewall" "allow_egress" {
  name    = "gcrunner-allow-egress"
  network = google_compute_network.gcrunner.id

  allow {
    protocol = "all"
  }

  direction          = "EGRESS"
  destination_ranges = ["0.0.0.0/0"]
}
```

### Orchestrator: Update VM creation

In `orchestrator/vm.go`, replace the `NetworkInterfaces` block:

```go
// Before
NetworkInterfaces: []*computepb.NetworkInterface{
    {
        AccessConfigs: []*computepb.AccessConfig{
            {
                Name: proto.String("External NAT"),
                Type: proto.String("ONE_TO_ONE_NAT"),
            },
        },
    },
},

// After
NetworkInterfaces: []*computepb.NetworkInterface{
    {
        Subnetwork: proto.String(fmt.Sprintf(
            "projects/%s/regions/%s/subnetworks/gcrunner-runners",
            project, region,
        )),
    },
},
```

The `region` variable needs to be passed into `createInstance` (currently only `zone` is passed). Extract region from the zone or pass it as a separate parameter.

### Environment variables

The orchestrator needs to know the subnetwork name. Options:
- Hardcode `gcrunner-runners` (matches Terraform)
- Add `GCRUNNER_SUBNET` env var for flexibility

### Private Google Access

`private_ip_google_access = true` on the subnet is critical — without it, VMs can't reach GCP APIs (Secret Manager, GCS, Compute API) since they won't have public IPs.
