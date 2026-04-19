package initjob

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerswarm "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

type Runner struct {
	dockerClient *client.Client

	pollInterval time.Duration
}

func NewRunner(
	dockerClient *client.Client,
	pollInterval time.Duration,
) *Runner {
	return &Runner{
		dockerClient: dockerClient,
		pollInterval: pollInterval,
	}
}

func (r *Runner) WaitJob(ctx context.Context, serviceID, jobName string) error {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait init job %s: %w", jobName, ctx.Err())
		case <-ticker.C:
			tasks, err := r.dockerClient.TaskList(ctx, dockerswarm.TaskListOptions{
				Filters: filters.NewArgs(filters.Arg("service", serviceID)),
			})
			if err != nil {
				return fmt.Errorf("inspect init job %s status: %w", jobName, err)
			}
			if len(tasks) == 0 {
				continue
			}

			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
			})

			task := tasks[0]
			state := task.Status.State
			switch state {
			case dockerswarm.TaskStateNew,
				dockerswarm.TaskStateAllocated,
				dockerswarm.TaskStatePending,
				dockerswarm.TaskStateAssigned,
				dockerswarm.TaskStateAccepted,
				dockerswarm.TaskStatePreparing,
				dockerswarm.TaskStateReady,
				dockerswarm.TaskStateStarting,
				dockerswarm.TaskStateRunning:
				continue
			case dockerswarm.TaskStateComplete:
				return nil
			case dockerswarm.TaskStateFailed,
				dockerswarm.TaskStateRejected,
				dockerswarm.TaskStateShutdown,
				dockerswarm.TaskStateOrphaned,
				dockerswarm.TaskStateRemove:
				reason := strings.TrimSpace(task.Status.Err)
				if reason == "" {
					reason = strings.TrimSpace(task.Status.Message)
				}
				if reason == "" {
					reason = string(state)
				}

				return &JobFailedError{
					ID:     task.ID,
					Name:   jobName,
					Reason: reason,
					logs:   r.readLogs(ctx, serviceID),
				}
			}
		}
	}
}

func (r *Runner) readLogs(ctx context.Context, serviceID string) []string {
	reader, err := r.dockerClient.ServiceLogs(ctx, serviceID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		slog.WarnContext(ctx, "[initjob] failed to fetch logs", slog.Any("err", err))
		return []string{}
	}

	logs := make([]string, 0)

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}

	return logs
}
