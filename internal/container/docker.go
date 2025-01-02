package container

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
)

type DockerManager struct {
	client *client.Client
	logger zerolog.Logger
}

func NewDockerManager(logger zerolog.Logger) (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &DockerManager{
		client: cli,
		logger: logger,
	}, nil
}

func (m *DockerManager) ListContainers(ctx context.Context) ([]types.Container, error) {
	return m.client.ContainerList(ctx, types.ContainerListOptions{})
}

func (m *DockerManager) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.client.ContainerInspect(ctx, containerID)
}

func (m *DockerManager) WatchEvents(ctx context.Context) (<-chan events.Message, <-chan error) {
	return m.client.Events(ctx, types.EventsOptions{})
}
