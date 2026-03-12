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

RUNNER_TOKEN="%s"
REPO_URL="%s"
RUNNER_NAME="%s"
LABELS="%s"

# Create runner user
useradd -m -s /bin/bash runner
cd /home/runner

# Download latest runner
RUNNER_VERSION=$(curl -s https://api.github.com/repos/actions/runner/releases/latest | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')
curl -sL "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz" -o runner.tar.gz
tar xzf runner.tar.gz
rm runner.tar.gz

# Install dependencies
./bin/installdependencies.sh

# Configure as ephemeral runner
chown -R runner:runner /home/runner
sudo -u runner ./config.sh \
  --url "${REPO_URL}" \
  --token "${RUNNER_TOKEN}" \
  --name "${RUNNER_NAME}" \
  --labels "${LABELS}" \
  --ephemeral \
  --unattended

# Run
sudo -u runner ./run.sh

# Self-destruct after job completes
ZONE=$(curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone | awk -F/ '{print $NF}')
INSTANCE=$(curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/name)
gcloud compute instances delete "${INSTANCE}" --zone="${ZONE}" --quiet
`

func createRunnerVM(ctx context.Context, event WorkflowJobEvent, labels *RunnerLabels) error {
	owner := event.Repository.Owner.Login
	repo := event.Repository.Name
	repoFullName := event.Repository.FullName

	// Get registration token
	regToken, err := getRegistrationToken(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("get registration token: %w", err)
	}

	instanceName := fmt.Sprintf("gcrunner-%d-%d", event.WorkflowJob.RunID, event.WorkflowJob.ID)
	repoURL := fmt.Sprintf("https://github.com/%s", repoFullName)
	runnerLabels := strings.Join(event.WorkflowJob.Labels, ",")

	startupScript := fmt.Sprintf(startupScriptTemplate, regToken, repoURL, instanceName, runnerLabels)

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
	sourceImage := "projects/ubuntu-os-cloud/global/images/family/ubuntu-2404-lts-amd64"

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

func parseDiskSize(disk string) int64 {
	disk = strings.TrimSuffix(strings.ToLower(disk), "gb")
	var size int64
	fmt.Sscanf(disk, "%d", &size)
	if size < 10 {
		size = 50
	}
	return size
}
