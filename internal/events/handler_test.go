package events

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types/events"
	"github.com/rs/zerolog"
)

type mockHandler struct {
	HandleEventFn func(ctx context.Context, event events.Message) error
	HandleCount   int
}

func (m *mockHandler) HandleEvent(ctx context.Context, event events.Message) error {
	m.HandleCount++
	if m.HandleEventFn != nil {
		return m.HandleEventFn(ctx, event)
	}
	return nil
}

func TestEventManager_HandleEvent(t *testing.T) {
	tests := []struct {
		name        string
		eventType   EventType
		setupMock   func(*mockHandler)
		wantErr     bool
		wantHandled int
	}{
		{
			name:      "Success: Event is processed normally",
			eventType: EventStart,
			setupMock: func(m *mockHandler) {
				m.HandleEventFn = func(_ context.Context, _ events.Message) error {
					return nil
				}
			},
			wantErr:     false,
			wantHandled: 1,
		},
		{
			name:      "Error: Handler returns an error",
			eventType: EventStart,
			setupMock: func(m *mockHandler) {
				m.HandleEventFn = func(_ context.Context, _ events.Message) error {
					return errors.New("handler error")
				}
			},
			wantErr:     true,
			wantHandled: 1,
		},
		{
			name:      "Success: Unregistered event type is ignored",
			eventType: "unknown",
			setupMock: func(m *mockHandler) {
				m.HandleEventFn = func(_ context.Context, _ events.Message) error {
					return nil
				}
			},
			wantErr:     false,
			wantHandled: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare mock handler
			mockHandler := &mockHandler{}
			if tt.setupMock != nil {
				tt.setupMock(mockHandler)
			}

			// Create event manager
			manager := NewManager(zerolog.Nop())
			manager.RegisterHandler(EventStart, mockHandler)

			// Create test event
			testEvent := events.Message{
				Type:   "container",
				Action: string(tt.eventType),
				ID:     "test-container",
			}

			// Execute event processing
			err := manager.HandleEvent(context.Background(), testEvent)

			// Verify error
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleEvent() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify processing count
			if mockHandler.HandleCount != tt.wantHandled {
				t.Errorf("HandleEvent() handled count = %v, want %v", mockHandler.HandleCount, tt.wantHandled)
			}
		})
	}
}
