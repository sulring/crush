package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/crush/internal/proto"
)

const (
	CreatedEvent EventType = "created"
	UpdatedEvent EventType = "updated"
	DeletedEvent EventType = "deleted"
)

type Suscriber[T any] interface {
	Subscribe(context.Context) <-chan Event[T]
}

type (
	PayloadType = string

	Payload struct {
		Type    PayloadType     `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

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

const (
	PayloadTypeLSPEvent               PayloadType = "lsp_event"
	PayloadTypeMCPEvent               PayloadType = "mcp_event"
	PayloadTypePermissionRequest      PayloadType = "permission_request"
	PayloadTypePermissionNotification PayloadType = "permission_notification"
	PayloadTypeMessage                PayloadType = "message"
	PayloadTypeSession                PayloadType = "session"
	PayloadTypeFile                   PayloadType = "file"
	PayloadTypeAgentEvent             PayloadType = "agent_event"
)

func (t EventType) MarshalText() ([]byte, error) {
	return []byte(t), nil
}

func (t *EventType) UnmarshalText(data []byte) error {
	*t = EventType(data)
	return nil
}

func (e Event[T]) MarshalJSON() ([]byte, error) {
	type Alias Event[T]

	var (
		typ string
		bts []byte
		err error
	)
	switch any(e.Payload).(type) {
	case proto.LSPEvent:
		typ = "lsp_event"
		bts, err = json.Marshal(e.Payload)
	case proto.MCPEvent:
		typ = "mcp_event"
		bts, err = json.Marshal(e.Payload)
	case proto.PermissionRequest:
		typ = "permission_request"
		bts, err = json.Marshal(e.Payload)
	case proto.PermissionNotification:
		typ = "permission_notification"
		bts, err = json.Marshal(e.Payload)
	case proto.Message:
		typ = "message"
		bts, err = json.Marshal(e.Payload)
	case proto.Session:
		typ = "session"
		bts, err = json.Marshal(e.Payload)
	case proto.File:
		typ = "file"
		bts, err = json.Marshal(e.Payload)
	case proto.AgentEvent:
		typ = "agent_event"
		bts, err = json.Marshal(e.Payload)
	default:
		panic(fmt.Sprintf("unknown payload type: %T", e.Payload))
	}

	if err != nil {
		return nil, err
	}

	p, err := json.Marshal(&Payload{
		Type:    typ,
		Payload: bts,
	})
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&struct {
		Payload json.RawMessage `json:"payload"`
		*Alias
	}{
		Payload: json.RawMessage(p),
		Alias:   (*Alias)(&e),
	})

	// slog.Info("marshalled event", "event", fmt.Sprintf("%q", string(b)))

	return b, err
}

func (e *Event[T]) UnmarshalJSON(data []byte) error {
	// slog.Info("unmarshalling event", "data", fmt.Sprintf("%q", string(data)))

	type Alias Event[T]
	aux := &struct {
		Payload json.RawMessage `json:"payload"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	e.Type = aux.Type

	slog.Info("unmarshalled event payload", "aux", fmt.Sprintf("%q", aux.Payload))

	var wp Payload
	if err := json.Unmarshal(aux.Payload, &wp); err != nil {
		return err
	}

	var pl any
	switch wp.Type {
	case "lsp_event":
		var p proto.LSPEvent
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	case "mcp_event":
		var p proto.MCPEvent
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	case "permission_request":
		var p proto.PermissionRequest
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	case "permission_notification":
		var p proto.PermissionNotification
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	case "message":
		var p proto.Message
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	case "session":
		var p proto.Session
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	case "file":
		var p proto.File
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	case "agent_event":
		var p proto.AgentEvent
		if err := json.Unmarshal(wp.Payload, &p); err != nil {
			return err
		}
		pl = p
	default:
		panic(fmt.Sprintf("unknown payload type: %q", wp.Type))
	}

	e.Payload = T(pl.(T))

	return nil
}
