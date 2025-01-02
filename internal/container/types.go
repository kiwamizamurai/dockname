package container

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
)

type Manager interface {
	ListContainers(ctx context.Context) ([]types.Container, error)
	InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error)
	WatchEvents(ctx context.Context) (<-chan events.Message, <-chan error)
}

type Container struct {
	ID              string
	Name            string
	Port            string
	Labels          map[string]string
	Status          string
	Network         string
	NetworkSettings *types.NetworkSettings
}
