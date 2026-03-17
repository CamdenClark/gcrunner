package function

import (
	"context"
	"fmt"
	"os"
	"sync"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
)

var (
	ctClient     *cloudtasks.Client
	ctClientOnce sync.Once
)

func getCloudTasksClient(ctx context.Context) (*cloudtasks.Client, error) {
	var initErr error
	ctClientOnce.Do(func() {
		ctClient, initErr = cloudtasks.NewClient(ctx)
	})
	return ctClient, initErr
}

// enqueueTask creates a Cloud Tasks HTTP task targeting the Cloud Run service.
// The task name is derived from the jobID to provide deduplication — if GitHub
// retries a webhook, the same task name prevents double-enqueue.
func enqueueTask(ctx context.Context, path string, payload []byte, jobID int64) error {
	client, err := getCloudTasksClient(ctx)
	if err != nil {
		return fmt.Errorf("create cloud tasks client: %w", err)
	}

	queuePath := os.Getenv("CLOUD_TASKS_QUEUE")
	if queuePath == "" {
		return fmt.Errorf("CLOUD_TASKS_QUEUE environment variable not set")
	}

	cloudRunURL := os.Getenv("CLOUD_RUN_URL")
	if cloudRunURL == "" {
		return fmt.Errorf("CLOUD_RUN_URL environment variable not set")
	}

	tasksSAEmail := os.Getenv("CLOUD_TASKS_SA_EMAIL")
	if tasksSAEmail == "" {
		return fmt.Errorf("CLOUD_TASKS_SA_EMAIL environment variable not set")
	}

	taskName := fmt.Sprintf("%s/tasks/job-%d-%s", queuePath, jobID, path[1:]) // strip leading /

	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			Name: taskName,
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					Url:        cloudRunURL + path,
					HttpMethod: taskspb.HttpMethod_POST,
					Body:       payload,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
						OidcToken: &taskspb.OidcToken{
							ServiceAccountEmail: tasksSAEmail,
						},
					},
				},
			},
		},
	}

	_, err = client.CreateTask(ctx, req)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}
