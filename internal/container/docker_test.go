package container

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/rs/zerolog"
)

// mockDockerClient implements DockerClient interface for testing
type mockDockerClient struct {
	mockListFunc    func() ([]types.Container, error)
	mockInspectFunc func(string) (types.ContainerJSON, error)
	mockEventsFunc  func() (<-chan events.Message, <-chan error)
}

func (m *mockDockerClient) ContainerList(_ context.Context, _ types.ContainerListOptions) ([]types.Container, error) {
	return m.mockListFunc()
}

func (m *mockDockerClient) ContainerInspect(_ context.Context, containerID string) (types.ContainerJSON, error) {
	return m.mockInspectFunc(containerID)
}

func (m *mockDockerClient) Events(_ context.Context, _ types.EventsOptions) (<-chan events.Message, <-chan error) {
	return m.mockEventsFunc()
}

func TestDockerManager_GetContainerInfo(t *testing.T) {
	tests := []struct {
		name            string
		container       types.Container
		mockInspectFunc func(_ string) (types.ContainerJSON, error)
		want            *Info
		wantErr         bool
	}{
		{
			name: "web container",
			container: types.Container{
				ID:    "test-container-id",
				Names: []string{"/web"},
				Ports: []types.Port{
					{PrivatePort: 80},
				},
			},
			mockInspectFunc: func(_ string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						ID:   "test-container-id",
						Name: "/web",
					},
					Config: &container.Config{
						Labels: map[string]string{
							LabelComposeService: "web",
						},
					},
					NetworkSettings: &types.NetworkSettings{
						Networks: map[string]*network.EndpointSettings{
							"default": {
								IPAddress: "172.17.0.2",
							},
						},
					},
				}, nil
			},
			want: &Info{
				ID:          "test-container-id",
				Name:        "web",
				ServiceName: "web",
				TargetPort:  80,
				IPAddress:   "172.17.0.2",
			},
			wantErr: false,
		},
		{
			name: "Success: Container without labels",
			container: types.Container{
				ID:    "test-container-id",
				Names: []string{"/myproject_web_1"},
				Ports: []types.Port{
					{PrivatePort: 8080},
				},
			},
			mockInspectFunc: func(_ string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						ID:   "test-container-id",
						Name: "/myproject_web_1",
					},
					Config: &container.Config{},
					NetworkSettings: &types.NetworkSettings{
						Networks: map[string]*network.EndpointSettings{
							"default": {
								IPAddress: "172.17.0.3",
							},
						},
					},
				}, nil
			},
			want: &Info{
				ID:          "test-container-id",
				Name:        "myproject_web_1",
				ServiceName: "web",
				TargetPort:  8080,
				IPAddress:   "172.17.0.3",
			},
			wantErr: false,
		},
		{
			name: "Success: RabbitMQ with management UI port",
			container: types.Container{
				ID:    "test-rabbitmq-id",
				Names: []string{"/myproject_queue_1"},
				Ports: []types.Port{
					{PrivatePort: 5672, PublicPort: 5672},   // AMQP port
					{PrivatePort: 15672, PublicPort: 15672}, // Management UI port
				},
			},
			mockInspectFunc: func(_ string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						ID:   "test-rabbitmq-id",
						Name: "/myproject_queue_1",
					},
					Config: &container.Config{},
					NetworkSettings: &types.NetworkSettings{
						Networks: map[string]*network.EndpointSettings{
							"default": {
								IPAddress: "172.17.0.4",
							},
						},
					},
				}, nil
			},
			want: &Info{
				ID:          "test-rabbitmq-id",
				Name:        "myproject_queue_1",
				ServiceName: "queue",
				TargetPort:  15672, // Should select management UI port
				IPAddress:   "172.17.0.4",
			},
			wantErr: false,
		},
		{
			name: "Success: pgAdmin with internal port 80",
			container: types.Container{
				ID:    "test-pgadmin-id",
				Names: []string{"/myproject_dbadmin_1"},
				Ports: []types.Port{
					{PrivatePort: 80, PublicPort: 8080}, // Web UI port
				},
			},
			mockInspectFunc: func(_ string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						ID:   "test-pgadmin-id",
						Name: "/myproject_dbadmin_1",
					},
					Config: &container.Config{},
					NetworkSettings: &types.NetworkSettings{
						Networks: map[string]*network.EndpointSettings{
							"default": {
								IPAddress: "172.17.0.5",
							},
						},
					},
				}, nil
			},
			want: &Info{
				ID:          "test-pgadmin-id",
				Name:        "myproject_dbadmin_1",
				ServiceName: "dbadmin",
				TargetPort:  80, // Should select internal port 80
				IPAddress:   "172.17.0.5",
			},
			wantErr: false,
		},
		{
			name: "Success: Container with explicit port label",
			container: types.Container{
				ID:    "test-custom-id",
				Names: []string{"/myproject_custom_1"},
				Labels: map[string]string{
					"dockname.port": "3000",
				},
				Ports: []types.Port{
					{PrivatePort: 3000, PublicPort: 8080},
				},
			},
			mockInspectFunc: func(_ string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						ID:   "test-custom-id",
						Name: "/myproject_custom_1",
					},
					Config: &container.Config{
						Labels: map[string]string{
							"dockname.port": "3000",
						},
					},
					NetworkSettings: &types.NetworkSettings{
						Networks: map[string]*network.EndpointSettings{
							"default": {
								IPAddress: "172.17.0.6",
							},
						},
					},
				}, nil
			},
			want: &Info{
				ID:          "test-custom-id",
				Name:        "myproject_custom_1",
				ServiceName: "custom",
				TargetPort:  3000, // Should use labeled port
				IPAddress:   "172.17.0.6",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			manager := &DockerManager{
				client: &mockDockerClient{
					mockListFunc:    func() ([]types.Container, error) { return []types.Container{}, nil },
					mockInspectFunc: tt.mockInspectFunc,
					mockEventsFunc:  func() (<-chan events.Message, <-chan error) { return nil, nil },
				},
				logger: logger,
			}

			info, err := manager.GetContainerInfo(context.Background(), tt.container)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetContainerInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if info == nil {
					t.Error("GetContainerInfo() returned nil info")
					return
				}

				if info.ID != tt.want.ID {
					t.Errorf("GetContainerInfo() ID = %v, want %v", info.ID, tt.want.ID)
				}
				if info.ServiceName != tt.want.ServiceName {
					t.Errorf("GetContainerInfo() ServiceName = %v, want %v", info.ServiceName, tt.want.ServiceName)
				}
				if info.TargetPort != tt.want.TargetPort {
					t.Errorf("GetContainerInfo() TargetPort = %v, want %v", info.TargetPort, tt.want.TargetPort)
				}
				if info.IPAddress != tt.want.IPAddress {
					t.Errorf("GetContainerInfo() IPAddress = %v, want %v", info.IPAddress, tt.want.IPAddress)
				}
			}
		})
	}
}
