package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/docker/docker/api/types"
	dockerevents "github.com/docker/docker/api/types/events"
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
	manager := &Manager{
		containerManager: containerManager,
		eventManager:     eventManager,
		proxyHandler:     proxyHandler,
		config:           config,
		logger:           logger,
	}

	// Register event handlers
	eventManager.RegisterHandler(events.EventStart, manager)
	eventManager.RegisterHandler(events.EventStop, manager)
	eventManager.RegisterHandler(events.EventDie, manager)
	eventManager.RegisterHandler(events.EventKill, manager)

	return manager
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
			return nil
		case err := <-errChan:
			return fmt.Errorf("container events error: %w", err)
		case event := <-eventsChan:
			if err := m.eventManager.HandleEvent(ctx, event); err != nil {
				m.logger.Error().Err(err).
					Str("event_action", event.Action).
					Str("event_id", event.ID).
					Msg("Failed to handle container event")
			}
		}
	}
}

func (m *Manager) registerContainer(ctx context.Context, container types.Container) error {
	m.logger.Debug().
		Str("container_id", container.ID).
		Interface("labels", container.Labels).
		Msg("Registering container")

	containerInfo, err := m.containerManager.GetContainerInfo(ctx, container)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}

	// Add detailed debug logging
	m.logger.Debug().
		Str("container_id", containerInfo.ID).
		Str("service_name", containerInfo.ServiceName).
		Int("target_port", containerInfo.TargetPort).
		Str("ip_address", containerInfo.IPAddress).
		Msg("Container info retrieved")

	// Skip if no service name or target port
	if containerInfo.ServiceName == "" || containerInfo.TargetPort == 0 {
		m.logger.Debug().
			Str("container_id", containerInfo.ID).
			Str("service_name", containerInfo.ServiceName).
			Int("target_port", containerInfo.TargetPort).
			Msg("Skipping container registration: missing service name or target port")
		return nil
	}

	// Create domain from service name
	domain := fmt.Sprintf("%s.localhost", containerInfo.ServiceName)

	// Create target URL
	targetURL := fmt.Sprintf("http://%s:%d", containerInfo.IPAddress, containerInfo.TargetPort)
	proxyURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		m.logger.Error().Err(err).
			Str("domain", domain).
			Str("target", targetURL).
			Msg("Proxy error")
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}

	m.proxyHandler.AddRoute(domain, proxy)
	m.logger.Info().
		Str("domain", domain).
		Str("target", targetURL).
		Str("service_name", containerInfo.ServiceName).
		Int("target_port", containerInfo.TargetPort).
		Str("container_id", containerInfo.ID).
		Msg("Registered container route")

	return nil
}

// HandleEvent implements events.Handler interface
func (m *Manager) HandleEvent(ctx context.Context, event dockerevents.Message) error {
	if event.Type != "container" {
		return nil
	}

	container := types.Container{
		ID:     event.Actor.ID,
		Labels: event.Actor.Attributes,
	}

	switch events.EventType(event.Action) {
	case events.EventStart:
		if err := m.registerContainer(ctx, container); err != nil {
			m.logger.Error().Err(err).
				Str("container_id", container.ID).
				Msg("Failed to register container")
			return err
		}
	case events.EventStop, events.EventDie, events.EventKill:
		containerInfo, err := m.containerManager.GetContainerInfo(ctx, container)
		if err != nil {
			m.logger.Error().Err(err).
				Str("container_id", container.ID).
				Msg("Failed to get container info for removal")
			return err
		}
		domain := fmt.Sprintf("%s.localhost", containerInfo.ServiceName)
		m.proxyHandler.RemoveRoute(domain)
		m.logger.Info().
			Str("domain", domain).
			Msg("Removed container route")
	}
	return nil
}
