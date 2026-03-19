package function

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/protobuf/proto"
)

const startupScriptTemplate = `#!/bin/bash
set -euo pipefail

METADATA_URL="http://metadata.google.internal/computeMetadata/v1"
METADATA_HEADER="Metadata-Flavor: Google"

# Retrieve JIT config from instance metadata and delete it immediately
JIT_CONFIG=$(curl -sf -H "${METADATA_HEADER}" "${METADATA_URL}/instance/attributes/jit-config")
# Remove the metadata key so credentials are no longer queryable
curl -sf -X DELETE -H "${METADATA_HEADER}" \
  "${METADATA_URL}/instance/attributes/jit-config" || true

CACHE_BUCKET="%s"
REPO_OWNER="%s"
REPO_NAME="%s"

cd /home/runner

# Start cache server if bucket is configured
if [ -n "${CACHE_BUCKET}" ] && [ -x /usr/local/bin/cache-server ]; then
  /usr/local/bin/cache-server \
    -bucket "${CACHE_BUCKET}" \
    -owner "${REPO_OWNER}" \
    -repo "${REPO_NAME}" &
  for i in $(seq 1 10); do
    curl -sf http://localhost:8787/health && break
    sleep 0.5
  done
  export ACTIONS_RESULTS_URL="http://localhost:8787/"
  export ACTIONS_CACHE_SERVICE_V2=true
fi

# Run with JIT config (skips config.sh entirely)
sudo -u runner -E ./run.sh --jitconfig "${JIT_CONFIG}"
`

func createRunnerVM(ctx context.Context, event WorkflowJobEvent, labels *RunnerLabels) error {
	owner := event.Repository.Owner.Login
	repo := event.Repository.Name
	repoFullName := event.Repository.FullName

	instanceName := fmt.Sprintf("gcrunner-%d-%d", event.WorkflowJob.RunID, event.WorkflowJob.ID)

	// Generate JIT config (replaces registration token + config.sh)
	jitConfig, err := generateJITConfig(ctx, owner, repo, instanceName, event.WorkflowJob.Labels)
	if err != nil {
		return fmt.Errorf("generate JIT config: %w", err)
	}

	cacheBucket := os.Getenv("GCRUNNER_CACHE_BUCKET")
	startupScript := fmt.Sprintf(startupScriptTemplate, cacheBucket, owner, repo)

	region := os.Getenv("GCE_REGION")
	if region == "" {
		region = "us-central1"
	}

	project := os.Getenv("GCP_PROJECT")

	// Determine zones to try
	var zones []string
	if labels.Zone != "" {
		zones = strings.Split(labels.Zone, "+")
	} else {
		var zoneErr error
		zones, zoneErr = ListZones(ctx, project, region)
		if zoneErr != nil {
			log.Printf("Failed to discover zones for %s, using fallback: %v", region, zoneErr)
			zones = []string{region + "-a", region + "-b", region + "-c"}
		}
	}

	var lastErr error
	for _, zone := range zones {
		// Resolve machine type per zone if not exact
		machineType := labels.Machine
		if labels.MachineMode != "exact" {
			resolved, resolveErr := ResolveMachineType(ctx, project, zone, labels)
			if resolveErr != nil {
				log.Printf("Failed to resolve machine type in %s: %v, trying next zone", zone, resolveErr)
				lastErr = resolveErr
				continue
			}
			machineType = resolved
		}

		err := createInstance(ctx, instanceName, zone, machineType, labels, startupScript, jitConfig)
		if err == nil {
			log.Printf("Created VM %s in %s (type=%s) for %s", instanceName, zone, machineType, repoFullName)
			return nil
		}
		lastErr = err

		kind := classifyInsertError(err)
		switch kind {
		case insertErrorQuota, insertErrorFatal:
			return fmt.Errorf("failed to create VM in %s: %w", zone, err)
		default:
			log.Printf("Failed to create VM in %s: %v, trying next zone", zone, err)
		}
	}

	return fmt.Errorf("failed to create VM in any zone: %w", lastErr)
}

func createInstance(ctx context.Context, name, zone, machineType string, labels *RunnerLabels, startupScript, jitConfig string) error {
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("create compute client: %w", err)
	}
	defer client.Close()

	project := os.Getenv("GCP_PROJECT")
	machineType = fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType)
	sourceImage := resolveSourceImage(labels.Image)

	diskSizeGB := parseDiskSize(labels.Disk)

	instance := &computepb.Instance{
		Name:        proto.String(name),
		MachineType: proto.String(machineType),
		Disks: []*computepb.AttachedDisk{
			{
				AutoDelete: proto.Bool(true),
				Boot:       proto.Bool(true),
				InitializeParams: &computepb.AttachedDiskInitializeParams{
					SourceImage: proto.String(sourceImage),
					DiskSizeGb:  proto.Int64(diskSizeGB),
					DiskType:    proto.String(fmt.Sprintf("zones/%s/diskTypes/%s", zone, labels.DiskType)),
				},
			},
		},
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
		Metadata: &computepb.Metadata{
			Items: []*computepb.Items{
				{
					Key:   proto.String("startup-script"),
					Value: proto.String(startupScript),
				},
				{
					Key:   proto.String("jit-config"),
					Value: proto.String(jitConfig),
				},
			},
		},
		Labels: map[string]string{
			"gcrunner": "true",
		},
		ServiceAccounts: []*computepb.ServiceAccount{
			{
				Email: proto.String(fmt.Sprintf("gcrunner-runner@%s.iam.gserviceaccount.com", project)),
				Scopes: []string{
					"https://www.googleapis.com/auth/cloud-platform",
				},
			},
		},
	}

	// Set spot scheduling if requested
	if labels.Spot {
		instance.Scheduling = &computepb.Scheduling{
			ProvisioningModel:  proto.String("SPOT"),
			InstanceTerminationAction: proto.String("DELETE"),
		}
	}

	op, err := client.Insert(ctx, &computepb.InsertInstanceRequest{
		Project:          project,
		Zone:             zone,
		InstanceResource: instance,
	})
	if err != nil {
		return err
	}

	// Wait for the operation to complete
	return op.Wait(ctx)
}

func deleteRunnerVM(ctx context.Context, name string) error {
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("create compute client: %w", err)
	}
	defer client.Close()

	project := os.Getenv("GCP_PROJECT")
	region := os.Getenv("GCE_REGION")
	if region == "" {
		region = "us-central1"
	}
	zones, zoneErr := ListZones(ctx, project, region)
	if zoneErr != nil {
		// Fallback: deletion must not fail due to zone listing issues
		zones = []string{region + "-a", region + "-b", region + "-c"}
	}

	for _, zone := range zones {
		op, err := client.Delete(ctx, &computepb.DeleteInstanceRequest{
			Project:  project,
			Zone:     zone,
			Instance: name,
		})
		if err != nil {
			continue
		}
		if err := op.Wait(ctx); err != nil {
			continue
		}
		log.Printf("Deleted VM %s in %s", name, zone)
		return nil
	}

	log.Printf("VM %s not found in any zone, may have already been deleted", name)
	return nil
}

func resolveSourceImage(image string) string {
	imageProject := os.Getenv("GCRUNNER_IMAGE_PROJECT")
	if imageProject == "" {
		imageProject = "gcrunner-images"
	}
	imageMap := map[string]string{
		"ubuntu24-full-x64": "gcrunner-ubuntu2404-x64",
		"ubuntu22-full-x64": "gcrunner-ubuntu2204-x64",
	}
	if family, ok := imageMap[image]; ok {
		return fmt.Sprintf("projects/%s/global/images/family/%s", imageProject, family)
	}
	if strings.Contains(image, "/") {
		return image
	}
	return "projects/ubuntu-os-cloud/global/images/family/ubuntu-2404-lts-amd64"
}

func parseDiskSize(disk string) int64 {
	disk = strings.TrimSuffix(strings.ToLower(disk), "gb")
	var size int64
	fmt.Sscanf(disk, "%d", &size)
	if size < 10 {
		size = 50
	}
	return size
}

type insertErrorKind int

const (
	insertErrorRetryable insertErrorKind = iota
	insertErrorQuota
	insertErrorFatal
)

// classifyInsertError categorizes a VM creation error to decide whether to retry.
func classifyInsertError(err error) insertErrorKind {
	if err == nil {
		return insertErrorRetryable
	}
	msg := err.Error()
	if strings.Contains(msg, "QUOTA_EXCEEDED") {
		return insertErrorQuota
	}
	if strings.Contains(msg, "RESOURCE_NOT_FOUND") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "Permission") {
		return insertErrorFatal
	}
	return insertErrorRetryable
}
