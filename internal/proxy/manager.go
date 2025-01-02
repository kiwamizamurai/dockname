package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/kiwamizamurai/dockname/internal/container"
	"github.com/kiwamizamurai/dockname/internal/events"
	"github.com/kiwamizamurai/dockname/internal/proxy/handler"
	"github.com/rs/zerolog"
)

type Config struct {
	Port           string
	UpdateInterval time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		Port:           ":80",
		UpdateInterval: 10 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
	}
}

type Manager struct {
	containerManager container.Manager
	eventManager     *events.Manager
	proxyHandler     *handler.ProxyHandler
	config           *Config
	logger           zerolog.Logger
}

func NewManager(containerManager container.Manager, config *Config, logger zerolog.Logger) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	eventManager := events.NewManager(logger)
	proxyHandler := handler.NewProxyHandler(logger)

	return &Manager{
		containerManager: containerManager,
		eventManager:     eventManager,
		proxyHandler:     proxyHandler,
		config:           config,
		logger:           logger,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		m.logger.Info().Msg("Stopping proxy manager")
	}()

	if err := m.initializeContainers(ctx); err != nil {
		return fmt.Errorf("failed to detect initial containers: %w", err)
	}

	go func() {
		if err := m.watchContainers(ctx); err != nil {
			m.logger.Error().Err(err).Msg("failed to watch containers")
		}
	}()

	server := &http.Server{
		Addr:    m.config.Port,
		Handler: m.proxyHandler,
	}

	m.logger.Info().Str("port", m.config.Port).Msg("Starting HTTP server")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

func (m *Manager) initializeContainers(ctx context.Context) error {
	containers, err := m.containerManager.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	m.logger.Info().Int("container_count", len(containers)).Msg("Getting initial container list")
	for _, container := range containers {
		if err := m.registerContainer(ctx, container); err != nil {
			m.logger.Error().Err(err).Str("container_id", container.ID).Msg("failed to register container")
			return fmt.Errorf("failed to register container %s: %w", container.ID, err)
		}
	}
	return nil
}

func (m *Manager) watchContainers(ctx context.Context) error {
	eventsChan, errChan := m.containerManager.WatchEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-eventsChan:
			if event.Type != "container" {
				continue
			}

			if err := m.eventManager.HandleEvent(ctx, event); err != nil {
				m.logger.Error().Err(err).
					Str("container_id", event.ID).
					Str("action", event.Action).
					Msg("failed to handle container event")
			}
		case err := <-errChan:
			if err != nil {
				m.logger.Error().Err(err).Msg("error monitoring events")
				return fmt.Errorf("error monitoring events: %w", err)
			}
		}
	}
}

func (m *Manager) registerContainer(ctx context.Context, container types.Container) error {
	domain, ok := container.Labels["dockname.domain"]
	if !ok {
		m.logger.Info().
			Str("container_id", container.ID).
			Interface("labels", container.Labels).
			Msg("dockname.domain label not found, skipping container")
		return nil
	}

	port := container.Labels["dockname.port"]
	if port == "" {
		port = "80" // Default port
	}

	containerJSON, err := m.containerManager.InspectContainer(ctx, container.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	var containerIP string
	for networkName, network := range containerJSON.NetworkSettings.Networks {
		if network.IPAddress != "" {
			containerIP = network.IPAddress
			m.logger.Info().
				Str("container_id", container.ID).
				Str("network", networkName).
				Str("ip", containerIP).
				Msg("Got container IP address")
			break
		}
	}

	if containerIP == "" {
		return fmt.Errorf("container IP address not found: %s", container.ID)
	}

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%s", containerIP, port))
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	m.proxyHandler.AddRoute(domain, proxy)

	m.logger.Info().
		Str("container_id", container.ID).
		Str("domain", domain).
		Str("target", targetURL.String()).
		Msg("Registered container")

	return nil
}
