package pubsub

import "context"

const (
	CreatedEvent EventType = "created"
	UpdatedEvent EventType = "updated"
	DeletedEvent EventType = "deleted"
)

type Suscriber[T any] interface {
	Subscribe(context.Context) <-chan Event[T]
}

type (
	// EventType identifies the type of event
	EventType string

	// Event represents an event in the lifecycle of a resource
	Event[T any] struct {
		Type    EventType `json:"type"`
		Payload T         `json:"payload"`
	}

	Publisher[T any] interface {
		Publish(EventType, T)
	}
)

func (t EventType) MarshalText() ([]byte, error) {
	return []byte(t), nil
}

func (t *EventType) UnmarshalText(data []byte) error {
	*t = EventType(data)
	return nil
}
