package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/rs/zerolog"
)

type mockManager struct {
	ListContainersFn   func(ctx context.Context) ([]types.Container, error)
	InspectContainerFn func(ctx context.Context, containerID string) (types.ContainerJSON, error)
	WatchEventsFn      func(ctx context.Context) (<-chan events.Message, <-chan error)
}

func (m *mockManager) ListContainers(ctx context.Context) ([]types.Container, error) {
	if m.ListContainersFn != nil {
		return m.ListContainersFn(ctx)
	}
	return nil, nil
}

func (m *mockManager) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if m.InspectContainerFn != nil {
		return m.InspectContainerFn(ctx, containerID)
	}
	return types.ContainerJSON{}, nil
}

func (m *mockManager) WatchEvents(ctx context.Context) (<-chan events.Message, <-chan error) {
	if m.WatchEventsFn != nil {
		return m.WatchEventsFn(ctx)
	}
	msgChan := make(chan events.Message)
	errChan := make(chan error)
	close(msgChan)
	close(errChan)
	return msgChan, errChan
}

func TestManager_Start(t *testing.T) {
	testContainers := []types.Container{
		{
			ID: "container1",
			Labels: map[string]string{
				"dockname.domain": "test1.example.com",
				"dockname.port":   "8080",
			},
		},
		{
			ID: "container2",
			Labels: map[string]string{
				"dockname.domain": "test2.example.com",
				"dockname.port":   "8081",
			},
		},
	}

	mockManager := &mockManager{
		ListContainersFn: func(_ context.Context) ([]types.Container, error) {
			return testContainers, nil
		},
		InspectContainerFn: func(_ context.Context, _ string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"bridge": {
							IPAddress: "172.17.0.2",
						},
					},
				},
			}, nil
		},
		WatchEventsFn: func(ctx context.Context) (<-chan events.Message, <-chan error) {
			eventsChan := make(chan events.Message)
			errChan := make(chan error)

			go func() {
				defer close(eventsChan)
				defer close(errChan)

				// Send test event
				select {
				case <-ctx.Done():
					return
				case eventsChan <- events.Message{
					Type:   "container",
					Action: "start",
					ID:     "container3",
				}:
				case <-time.After(100 * time.Millisecond):
					return
				}
			}()

			return eventsChan, errChan
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	config := &Config{
		Port:           ":8080",
		UpdateInterval: 100 * time.Millisecond,
	}
	manager := NewManager(mockManager, config, zerolog.Nop())

	// Start the manager in a goroutine
	go func() {
		if err := manager.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Manager.Start() error = %v", err)
		}
	}()

	// Wait for a short period to allow the manager to process events
	time.Sleep(200 * time.Millisecond)

	// Cancel the context to stop the manager
	cancel()
}

func TestManager_initializeContainers(t *testing.T) {
	tests := []struct {
		name       string
		containers []types.Container
		setupMock  func(*mockManager)
		wantErr    bool
	}{
		{
			name: "Success: Containers initialize correctly",
			containers: []types.Container{
				{
					ID: "container1",
					Labels: map[string]string{
						"dockname.domain": "test.example.com",
						"dockname.port":   "8080",
					},
				},
			},
			setupMock: func(m *mockManager) {
				m.ListContainersFn = func(_ context.Context) ([]types.Container, error) {
					return []types.Container{
						{
							ID: "container1",
							Labels: map[string]string{
								"dockname.domain": "test.example.com",
								"dockname.port":   "8080",
							},
						},
					}, nil
				}
				m.InspectContainerFn = func(_ context.Context, _ string) (types.ContainerJSON, error) {
					return types.ContainerJSON{
						NetworkSettings: &types.NetworkSettings{
							Networks: map[string]*network.EndpointSettings{
								"bridge": {
									IPAddress: "172.17.0.2",
								},
							},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Error: Container inspection fails",
			containers: []types.Container{
				{
					ID: "container1",
					Labels: map[string]string{
						"dockname.domain": "test.example.com",
						"dockname.port":   "8080",
					},
				},
			},
			setupMock: func(m *mockManager) {
				m.ListContainersFn = func(_ context.Context) ([]types.Container, error) {
					return []types.Container{
						{
							ID: "container1",
							Labels: map[string]string{
								"dockname.domain": "test.example.com",
								"dockname.port":   "8080",
							},
						},
					}, nil
				}
				m.InspectContainerFn = func(_ context.Context, _ string) (types.ContainerJSON, error) {
					return types.ContainerJSON{}, errors.New("inspect error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &mockManager{}
			if tt.setupMock != nil {
				tt.setupMock(mockManager)
			}

			manager := NewManager(mockManager, nil, zerolog.Nop())

			err := manager.initializeContainers(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("initializeContainers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
