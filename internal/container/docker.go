package container

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
)

type DockerManager struct {
	client DockerClient
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

func (m *DockerManager) GetContainerInfo(ctx context.Context, container types.Container) (*Info, error) {
	// Get detailed container information
	containerJSON, err := m.InspectContainer(ctx, container.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Get service name from labels or container name
	serviceName := container.Labels[LabelComposeService]
	if serviceName == "" {
		// If no service name label, use container name without project prefix
		name := strings.TrimPrefix(containerJSON.Name, "/")
		parts := strings.Split(name, "_")
		if len(parts) > 1 {
			serviceName = parts[1] // Usually format is: project_service_1
		} else {
			serviceName = name
		}
	}

	// Log container ports for debugging
	m.logger.Debug().
		Str("container", containerJSON.Name).
		Str("service", serviceName).
		Interface("ports", container.Ports).
		Interface("exposed_ports", containerJSON.Config.ExposedPorts).
		Interface("labels", container.Labels).
		Msg("Container information")

	// Detect target port
	var targetPort int
	if portLabel := container.Labels["dockname.port"]; portLabel != "" {
		// If port is explicitly specified via label, use it
		if port, err := strconv.Atoi(portLabel); err == nil {
			targetPort = port
			m.logger.Debug().
				Str("container", containerJSON.Name).
				Int("target_port", targetPort).
				Msg("Selected target port from label")
		}
	}

	// If no port label, find the first published port
	if targetPort == 0 && len(container.Ports) > 0 {
		// First, try to find a published HTTP port (80, 8080, 15672, etc.)
		httpPorts := map[uint16]bool{
			80:    true, // HTTP
			443:   true, // HTTPS
			8080:  true, // Alternative HTTP
			8443:  true, // Alternative HTTPS
			3000:  true, // React, Node.js apps
			4200:  true, // Angular
			5000:  true, // Flask, ASP.NET
			5173:  true, // Vite
			8000:  true, // Django, PHP
			8081:  true, // Alternative web
			8888:  true, // Jupyter Notebook
			9000:  true, // PHP-FPM, Portainer
			15672: true, // RabbitMQ management UI
			27017: true, // MongoDB
			6379:  true, // Redis
			5432:  true, // PostgreSQL
			3306:  true, // MySQL
			1337:  true, // Strapi
			9090:  true, // Prometheus
			3001:  true, // Development servers
			8082:  true, // Development servers
			8083:  true, // Development servers
			8084:  true, // Development servers
			8085:  true, // Development servers
			2375:  true, // Docker API
			2376:  true, // Docker API (TLS)
			3100:  true, // Grafana Loki
			5601:  true, // Kibana
			7474:  true, // Neo4j Browser
			8025:  true, // MailHog UI
			8086:  true, // InfluxDB UI
			8200:  true, // Vault UI
			8529:  true, // ArangoDB UI
			9001:  true, // Supervisor
			9092:  true, // Kafka
			9200:  true, // Elasticsearch
			9443:  true, // Rancher
			10000: true, // Webmin
			26257: true, // CockroachDB UI
		}
		for _, port := range container.Ports {
			if port.PublicPort > 0 {
				if httpPorts[port.PrivatePort] {
					targetPort = int(port.PrivatePort)
					m.logger.Debug().
						Str("container", containerJSON.Name).
						Int("target_port", targetPort).
						Uint16("public_port", port.PublicPort).
						Msg("Selected target port from HTTP ports")
					break
				} else if port.PrivatePort == 80 {
					// If container exposes port 80 internally, prioritize it
					targetPort = 80
					m.logger.Debug().
						Str("container", containerJSON.Name).
						Int("target_port", targetPort).
						Uint16("public_port", port.PublicPort).
						Msg("Selected internal HTTP port 80")
					break
				}
			}
		}

		// If no HTTP port found, use the first published port
		if targetPort == 0 {
			for _, port := range container.Ports {
				if port.PublicPort > 0 {
					targetPort = int(port.PrivatePort)
					m.logger.Debug().
						Str("container", containerJSON.Name).
						Int("target_port", targetPort).
						Uint16("public_port", port.PublicPort).
						Msg("Selected target port from first published port")
					break
				}
			}
		}

		// If still no port found, use the first available port
		if targetPort == 0 {
			targetPort = int(container.Ports[0].PrivatePort)
			m.logger.Debug().
				Str("container", containerJSON.Name).
				Int("target_port", targetPort).
				Msg("Selected target port from first available port")
		}
	}

	// Get container IP address
	var ipAddress string
	if containerJSON.NetworkSettings != nil {
		for networkName, network := range containerJSON.NetworkSettings.Networks {
			ipAddress = network.IPAddress
			m.logger.Debug().
				Str("container", containerJSON.Name).
				Str("network", networkName).
				Str("ip", ipAddress).
				Msg("Selected container network")
			break
		}
	}

	// Log final container info
	m.logger.Debug().
		Str("container", containerJSON.Name).
		Str("service", serviceName).
		Int("port", targetPort).
		Str("ip", ipAddress).
		Msg("Container info resolved")

	return &Info{
		ID:          container.ID,
		Name:        strings.TrimPrefix(containerJSON.Name, "/"),
		ServiceName: serviceName,
		TargetPort:  targetPort,
		IPAddress:   ipAddress,
	}, nil
}
