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

JIT_CONFIG="%s"
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
	startupScript := fmt.Sprintf(startupScriptTemplate, jitConfig, cacheBucket, owner, repo)

	region := os.Getenv("GCE_REGION")
	if region == "" {
		region = "us-central1"
	}

	// Try each zone in the region
	zones := []string{region + "-a", region + "-b", region + "-c", region + "-f"}

	var lastErr error
	for _, zone := range zones {
		err := createInstance(ctx, instanceName, zone, labels, startupScript)
		if err == nil {
			log.Printf("Created VM %s in %s for %s", instanceName, zone, repoFullName)
			return nil
		}
		lastErr = err
		log.Printf("Failed to create VM in %s: %v, trying next zone", zone, err)
	}

	return fmt.Errorf("failed to create VM in any zone: %w", lastErr)
}

func createInstance(ctx context.Context, name, zone string, labels *RunnerLabels, startupScript string) error {
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("create compute client: %w", err)
	}
	defer client.Close()

	project := os.Getenv("GCP_PROJECT")
	machineType := fmt.Sprintf("zones/%s/machineTypes/%s", zone, labels.Machine)
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
					DiskType:    proto.String(fmt.Sprintf("zones/%s/diskTypes/pd-balanced", zone)),
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

	zones := []string{region + "-a", region + "-b", region + "-c", region + "-f"}

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
