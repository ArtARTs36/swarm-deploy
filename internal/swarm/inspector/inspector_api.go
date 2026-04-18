package inspector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	dockerswarm "github.com/docker/docker/api/types/swarm"
)

const (
	defaultServiceLogsLimit           = 200
	dockerLogFrameHeaderSize          = 8
	serviceLogsScannerInitialBufSize  = 64 * 1024
	serviceLogsScannerMaxTokenBufSize = 1 << 20
)

func (i *Inspector) InspectServiceStatus(ctx context.Context, stackName, serviceName string) (ServiceStatus, error) {
	fullServiceName := fmt.Sprintf("%s_%s", stackName, serviceName)
	service, _, err := i.dockerClient.ServiceInspectWithRaw(ctx, fullServiceName, dockerswarm.ServiceInspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ServiceStatus{}, ErrServiceNotFound
		}
		return ServiceStatus{}, fmt.Errorf("inspect service %s: %w", fullServiceName, err)
	}

	status := ServiceStatus{
		Stack:   stackName,
		Service: serviceName,
	}
	if service.Spec.TaskTemplate.ContainerSpec != nil {
		status.Image = service.Spec.TaskTemplate.ContainerSpec.Image
	}

	if resources := service.Spec.TaskTemplate.Resources; resources != nil && resources.Reservations != nil {
		status.RequestedRAMBytes = resources.Reservations.MemoryBytes
		status.RequestedCPUNano = resources.Reservations.NanoCPUs
	}
	if resources := service.Spec.TaskTemplate.Resources; resources != nil && resources.Limits != nil {
		status.LimitRAMBytes = resources.Limits.MemoryBytes
		status.LimitCPUNano = resources.Limits.NanoCPUs
	}

	return status, nil
}

// InspectServiceLabels returns service, container and image labels for a stack service.
func (i *Inspector) InspectServiceLabels(
	ctx context.Context,
	stackName, serviceName string,
) (ServiceLabels, error) {
	fullServiceName := fmt.Sprintf("%s_%s", stackName, serviceName)
	service, _, err := i.dockerClient.ServiceInspectWithRaw(ctx, fullServiceName, dockerswarm.ServiceInspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return ServiceLabels{}, ErrServiceNotFound
		}
		return ServiceLabels{}, fmt.Errorf("inspect service %s: %w", fullServiceName, err)
	}

	labels := ServiceLabels{
		Service: cloneStringMap(service.Spec.Labels),
	}

	containerSpec := service.Spec.TaskTemplate.ContainerSpec
	if containerSpec != nil {
		labels.Container = cloneStringMap(containerSpec.Labels)
		labels.ContainerEnv = cloneStringSlice(containerSpec.Env)
	}

	imageRef := ""
	if containerSpec != nil {
		imageRef = containerSpec.Image
	}

	slog.DebugContext(ctx, "[swarm] inspecting image", slog.String("image_ref", imageRef))

	image, err := i.dockerClient.ImageInspect(ctx, imageRef)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			slog.DebugContext(ctx, "[swarm] image not found", slog.String("image_ref", imageRef))

			return labels, nil
		}
		return labels, fmt.Errorf("inspect image %s: %w", imageRef, err)
	}

	slog.DebugContext(ctx, "[swarm] image inspected",
		slog.String("image_ref", imageRef),
		slog.Any("image", image),
	)

	if image.Config != nil {
		labels.Image = cloneStringMap(image.Config.Labels)
	}
	return labels, nil
}

// ServiceLogsOptions configures stack service logs query.
type ServiceLogsOptions struct {
	// Limit is max number of latest lines to return.
	Limit int
	// Since defines lower bound for log timestamps.
	Since *time.Time
	// Until defines upper bound for log timestamps.
	Until *time.Time
}

// InspectServiceLogs returns recent logs for a stack service.
func (i *Inspector) InspectServiceLogs(
	ctx context.Context,
	stackName string,
	serviceName string,
	options ServiceLogsOptions,
) ([]string, error) {
	fullServiceName := fmt.Sprintf("%s_%s", stackName, serviceName)
	reader, err := i.dockerClient.ServiceLogs(ctx, fullServiceName, buildDockerServiceLogsOptions(options))
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil, ErrServiceNotFound
		}

		return nil, fmt.Errorf("read logs for service %s: %w", fullServiceName, err)
	}
	defer reader.Close()

	rawLogs, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read logs stream for service %s: %w", fullServiceName, err)
	}

	decodedLogs := demultiplexDockerLogStream(rawLogs)

	logs := make([]string, 0)

	scanner := bufio.NewScanner(bytes.NewReader(decodedLogs))
	scanner.Buffer(
		make([]byte, 0, serviceLogsScannerInitialBufSize),
		serviceLogsScannerMaxTokenBufSize,
	)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		logs = append(logs, line)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("scan logs for service %s: %w", fullServiceName, scanErr)
	}

	return logs, nil
}

func buildDockerServiceLogsOptions(options ServiceLogsOptions) container.LogsOptions {
	limit := options.Limit
	if limit <= 0 {
		limit = defaultServiceLogsLimit
	}

	logsOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       strconv.Itoa(limit),
	}

	if options.Since != nil {
		logsOptions.Since = options.Since.UTC().Format(time.RFC3339Nano)
	}
	if options.Until != nil {
		logsOptions.Until = options.Until.UTC().Format(time.RFC3339Nano)
	}

	return logsOptions
}

func demultiplexDockerLogStream(raw []byte) []byte {
	// Docker multiplexed logs use 8-byte frame headers:
	// [stream(1)][0][0][0][payload_size_be_uint32].
	// If payload is shorter than a single header, treat it as plain text.
	if len(raw) < dockerLogFrameHeaderSize {
		return raw
	}

	decoded := bytes.NewBuffer(make([]byte, 0, len(raw)))
	cursor := 0
	parsedFrames := false

	for cursor+dockerLogFrameHeaderSize <= len(raw) {
		header := raw[cursor : cursor+dockerLogFrameHeaderSize]
		// Non-zero reserved bytes mean this is not a Docker multiplexed frame.
		// If this happens before any parsed frame, keep stream untouched.
		// If we already parsed something, append the remainder as best-effort.
		if header[1] != 0 || header[2] != 0 || header[3] != 0 {
			if !parsedFrames {
				return raw
			}

			decoded.Write(raw[cursor:])
			return decoded.Bytes()
		}

		frameSize := int(binary.BigEndian.Uint32(header[4:dockerLogFrameHeaderSize]))
		cursor += dockerLogFrameHeaderSize

		// Broken frame length: keep behavior safe and non-destructive.
		if frameSize < 0 || cursor+frameSize > len(raw) {
			if !parsedFrames {
				return raw
			}

			decoded.Write(raw[cursor-dockerLogFrameHeaderSize:])
			return decoded.Bytes()
		}

		if frameSize > 0 {
			decoded.Write(raw[cursor : cursor+frameSize])
		}
		cursor += frameSize
		parsedFrames = true
	}

	// Preserve trailing bytes that don't form a full header.
	if cursor < len(raw) {
		decoded.Write(raw[cursor:])
	}
	// If we did not recognize a single frame, return original stream.
	if !parsedFrames {
		return raw
	}

	return decoded.Bytes()
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	out := make([]string, len(in))
	copy(out, in)

	return out
}
