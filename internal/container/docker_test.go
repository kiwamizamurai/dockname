package container

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestNewDockerManager(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Success: Create new DockerManager",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			manager, err := NewDockerManager(logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDockerManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && manager == nil {
				t.Error("NewDockerManager() returned nil manager")
			}
		})
	}
}

func TestDockerManager_ListContainers(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*DockerManager)
		wantErr bool
	}{
		{
			name:    "Success: List containers",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			manager, err := NewDockerManager(logger)
			if err != nil {
				t.Fatalf("Failed to create DockerManager: %v", err)
			}

			if tt.setup != nil {
				tt.setup(manager)
			}

			containers, err := manager.ListContainers(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ListContainers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && containers == nil {
				t.Error("ListContainers() returned nil containers slice")
			}
		})
	}
}

func TestDockerManager_InspectContainer(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		setup       func(*DockerManager)
		wantErr     bool
	}{
		{
			name:        "Error: Invalid container ID",
			containerID: "invalid-container-id",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			manager, err := NewDockerManager(logger)
			if err != nil {
				t.Fatalf("Failed to create DockerManager: %v", err)
			}

			if tt.setup != nil {
				tt.setup(manager)
			}

			container, err := manager.InspectContainer(context.Background(), tt.containerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("InspectContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && container.ID == "" {
				t.Error("InspectContainer() returned empty container ID")
			}
		})
	}
}

func TestDockerManager_WatchEvents(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*DockerManager)
		wantErr bool
	}{
		{
			name:    "Success: Watch events",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			manager, err := NewDockerManager(logger)
			if err != nil {
				t.Fatalf("Failed to create DockerManager: %v", err)
			}

			if tt.setup != nil {
				tt.setup(manager)
			}

			events, errs := manager.WatchEvents(context.Background())
			if events == nil {
				t.Error("WatchEvents() returned nil events channel")
			}
			if errs == nil {
				t.Error("WatchEvents() returned nil errors channel")
			}
		})
	}
}
