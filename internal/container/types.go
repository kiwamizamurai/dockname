package container

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
)

// DockerClient defines the interface for Docker operations
type DockerClient interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error)
}

type Info struct {
	ID          string
	Name        string
	ServiceName string
	TargetPort  int
	IPAddress   string
}

// Manager interface defines the methods required for container management
type Manager interface {
	ListContainers(ctx context.Context) ([]types.Container, error)
	InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error)
	WatchEvents(ctx context.Context) (<-chan events.Message, <-chan error)
	GetContainerInfo(ctx context.Context, container types.Container) (*Info, error)
}

const (
	LabelComposeService = "com.docker.compose.service"
	LabelComposeProject = "com.docker.compose.project"
)

type Container struct {
	ID              string
	Name            string
	Port            string
	Labels          map[string]string
	Status          string
	Network         string
	NetworkSettings *types.NetworkSettings
}
