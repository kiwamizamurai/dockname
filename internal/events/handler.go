package events

import (
	"context"

	"github.com/docker/docker/api/types/events"
	"github.com/rs/zerolog"
)

type Handler interface {
	HandleEvent(ctx context.Context, event events.Message) error
}

type EventType string

const (
	EventStart EventType = "start"
	EventStop  EventType = "stop"
	EventDie   EventType = "die"
	EventKill  EventType = "kill"
)

type Manager struct {
	handlers map[EventType][]Handler
	logger   zerolog.Logger
}

func NewManager(logger zerolog.Logger) *Manager {
	return &Manager{
		handlers: make(map[EventType][]Handler),
		logger:   logger,
	}
}

func (m *Manager) RegisterHandler(eventType EventType, handler Handler) {
	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

func (m *Manager) HandleEvent(ctx context.Context, event events.Message) error {
	eventType := EventType(event.Action)
	handlers, exists := m.handlers[eventType]
	if !exists {
		return nil
	}

	for _, handler := range handlers {
		if err := handler.HandleEvent(ctx, event); err != nil {
			m.logger.Error().Err(err).
				Str("event_type", string(eventType)).
				Str("event_id", event.ID).
				Msg("Failed to process event")
			return err
		}
	}
	return nil
}
