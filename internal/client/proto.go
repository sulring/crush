package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
)

func (c *Client) SubscribeEvents(ctx context.Context) (<-chan any, error) {
	events := make(chan any, 100)
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/events", c.id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	r.Header.Set("Accept", "text/event-stream")
	r.Header.Set("Cache-Control", "no-cache")
	r.Header.Set("Connection", "keep-alive")

	go func() {
		rsp, err := c.h.Do(r)
		if err != nil {
			slog.Error("subscribing to events", "error", err)
			return
		}

		defer rsp.Body.Close()

		if rsp.StatusCode != http.StatusOK {
			slog.Error("subscribing to events", "status_code", rsp.StatusCode)
			return
		}

		scr := bufio.NewReader(rsp.Body)
		for {
			line, err := scr.ReadBytes('\n')
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				slog.Error("reading from events stream", "error", err)
				time.Sleep(time.Second * 2)
				continue
			}
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				// End of an event
				continue
			}

			data, ok := bytes.CutPrefix(line, []byte("data:"))
			if !ok {
				slog.Warn("invalid event format", "line", string(line))
				continue
			}

			data = bytes.TrimSpace(data)

			var event pubsub.Event[any]
			if err := json.Unmarshal(data, &event); err != nil {
				slog.Error("unmarshaling event", "error", err)
				continue
			}

			type alias pubsub.Event[any]
			aux := &struct {
				Payload json.RawMessage `json:"payload"`
				*alias
			}{
				alias: (*alias)(&event),
			}

			if err := json.Unmarshal(data, &aux); err != nil {
				slog.Error("unmarshaling event payload", "error", err)
				continue
			}

			var p pubsub.Payload
			if err := json.Unmarshal(aux.Payload, &p); err != nil {
				slog.Error("unmarshaling event payload", "error", err)
				continue
			}

			switch p.Type {
			case pubsub.PayloadTypeLSPEvent:
				var e pubsub.Event[proto.LSPEvent]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			case pubsub.PayloadTypeMCPEvent:
				var e pubsub.Event[proto.MCPEvent]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			case pubsub.PayloadTypePermissionRequest:
				var e pubsub.Event[proto.PermissionRequest]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			case pubsub.PayloadTypePermissionNotification:
				var e pubsub.Event[proto.PermissionNotification]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			case pubsub.PayloadTypeMessage:
				var e pubsub.Event[proto.Message]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			case pubsub.PayloadTypeSession:
				var e pubsub.Event[proto.Session]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			case pubsub.PayloadTypeFile:
				var e pubsub.Event[proto.File]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			case pubsub.PayloadTypeAgentEvent:
				var e pubsub.Event[proto.AgentEvent]
				_ = json.Unmarshal(data, &e)
				sendEvent(ctx, events, e)
			default:
				slog.Warn("unknown event type", "type", p.Type)
				continue
			}
		}
	}()

	return events, nil
}

func sendEvent(ctx context.Context, evc chan any, ev any) {
	slog.Info("event received", "event", fmt.Sprintf("%T %+v", ev, ev))
	select {
	case evc <- ev:
	case <-ctx.Done():
		close(evc)
		return
	}
}

func (c *Client) GetLSPDiagnostics(ctx context.Context, lsp string) (map[protocol.DocumentURI][]protocol.Diagnostic, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/lsps/%s/diagnostics", c.id, lsp), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get LSP diagnostics: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get LSP diagnostics: status code %d", rsp.StatusCode)
	}
	var diagnostics map[protocol.DocumentURI][]protocol.Diagnostic
	if err := json.NewDecoder(rsp.Body).Decode(&diagnostics); err != nil {
		return nil, fmt.Errorf("failed to decode LSP diagnostics: %w", err)
	}
	return diagnostics, nil
}

func (c *Client) GetLSPs(ctx context.Context) (map[string]app.LSPClientInfo, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/lsps", c.id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get LSPs: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get LSPs: status code %d", rsp.StatusCode)
	}
	var lsps map[string]app.LSPClientInfo
	if err := json.NewDecoder(rsp.Body).Decode(&lsps); err != nil {
		return nil, fmt.Errorf("failed to decode LSPs: %w", err)
	}
	return lsps, nil
}

func (c *Client) GetAgentSessionQueuedPrompts(ctx context.Context, sessionID string) (int, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/agent/sessions/%s/prompts/queued", c.id, sessionID), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return 0, fmt.Errorf("failed to get session agent queued prompts: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get session agent queued prompts: status code %d", rsp.StatusCode)
	}
	var count int
	if err := json.NewDecoder(rsp.Body).Decode(&count); err != nil {
		return 0, fmt.Errorf("failed to decode session agent queued prompts: %w", err)
	}
	return count, nil
}

func (c *Client) ClearAgentSessionQueuedPrompts(ctx context.Context, sessionID string) error {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/agent/sessions/%s/prompts/clear", c.id, sessionID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to clear session agent queued prompts: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to clear session agent queued prompts: status code %d", rsp.StatusCode)
	}
	return nil
}

func (c *Client) GetAgentInfo(ctx context.Context) (*proto.AgentInfo, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/agent", c.id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent status: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get agent status: status code %d", rsp.StatusCode)
	}
	var info proto.AgentInfo
	if err := json.NewDecoder(rsp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode agent status: %w", err)
	}
	return &info, nil
}

func (c *Client) UpdateAgent(ctx context.Context) error {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/agent/update", c.id), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update agent: status code %d", rsp.StatusCode)
	}
	return nil
}

func (c *Client) SendMessage(ctx context.Context, sessionID, message string, attchments ...message.Attachment) error {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/agent", c.id), jsonBody(proto.AgentMessage{
		SessionID:   sessionID,
		Prompt:      message,
		Attachments: attchments,
	}))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to send message to agent: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message to agent: status code %d", rsp.StatusCode)
	}
	return nil
}

func (c *Client) GetAgentSessionInfo(ctx context.Context, sessionID string) (*proto.AgentSession, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/agent/sessions/%s", c.id, sessionID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get session agent info: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get session agent info: status code %d", rsp.StatusCode)
	}
	var info proto.AgentSession
	if err := json.NewDecoder(rsp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode session agent info: %w", err)
	}
	return &info, nil
}

func (c *Client) AgentSummarizeSession(ctx context.Context, sessionID string) error {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/agent/sessions/%s/summarize", c.id, sessionID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to summarize session: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to summarize session: status code %d", rsp.StatusCode)
	}
	return nil
}

func (c *Client) ListMessages(ctx context.Context, sessionID string) ([]message.Message, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/sessions/%s/messages", c.id, sessionID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get messages: status code %d", rsp.StatusCode)
	}
	var messages []message.Message
	if err := json.NewDecoder(rsp.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}
	return messages, nil
}

func (c *Client) GetSession(ctx context.Context, sessionID string) (*session.Session, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/sessions/%s", c.id, sessionID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get session: status code %d", rsp.StatusCode)
	}
	var sess session.Session
	if err := json.NewDecoder(rsp.Body).Decode(&sess); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}
	return &sess, nil
}

func (c *Client) InitiateAgentProcessing(ctx context.Context) error {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/agent/init", c.id), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to initiate session agent processing: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to initiate session agent processing: status code %d", rsp.StatusCode)
	}
	return nil
}

func (c *Client) ListSessionHistoryFiles(ctx context.Context, sessionID string) ([]history.File, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/sessions/%s/history", c.id, sessionID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get session history files: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get session history files: status code %d", rsp.StatusCode)
	}
	var files []history.File
	if err := json.NewDecoder(rsp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to decode session history files: %w", err)
	}
	return files, nil
}

func (c *Client) CreateSession(ctx context.Context, title string) (*session.Session, error) {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/sessions", c.id), jsonBody(session.Session{Title: title}))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create session: status code %d", rsp.StatusCode)
	}
	var sess session.Session
	if err := json.NewDecoder(rsp.Body).Decode(&sess); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}
	return &sess, nil
}

func (c *Client) ListSessions(ctx context.Context) ([]session.Session, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/sessions", c.id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get sessions: status code %d", rsp.StatusCode)
	}
	var sessions []session.Session
	if err := json.NewDecoder(rsp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}
	return sessions, nil
}

func (c *Client) GrantPermission(ctx context.Context, req proto.PermissionGrant) error {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/permissions/grant", c.id), jsonBody(req))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to grant permission: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to grant permission: status code %d", rsp.StatusCode)
	}
	return nil
}

func (c *Client) SetPermissionsSkipRequests(ctx context.Context, skip bool) error {
	r, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost/v1/instances/%s/permissions/skip", c.id), jsonBody(proto.PermissionSkipRequest{Skip: skip}))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to set permissions skip requests: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set permissions skip requests: status code %d", rsp.StatusCode)
	}
	return nil
}

func (c *Client) GetPermissionsSkipRequests(ctx context.Context) (bool, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/permissions/skip", c.id), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return false, fmt.Errorf("failed to get permissions skip requests: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to get permissions skip requests: status code %d", rsp.StatusCode)
	}
	var skip proto.PermissionSkipRequest
	if err := json.NewDecoder(rsp.Body).Decode(&skip); err != nil {
		return false, fmt.Errorf("failed to decode permissions skip requests: %w", err)
	}
	return skip.Skip, nil
}

func (c *Client) GetConfig(ctx context.Context) (*config.Config, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost/v1/instances/%s/config", c.id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get config: status code %d", rsp.StatusCode)
	}
	var cfg config.Config
	if err := json.NewDecoder(rsp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	return &cfg, nil
}

func (c *Client) CreateInstance(ctx context.Context, ins proto.Instance) (*proto.Instance, error) {
	r, err := http.NewRequestWithContext(ctx, "POST", "http://localhost/v1/instances", jsonBody(ins))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	r.Header.Set("Content-Type", "application/json")
	rsp, err := c.h.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create instance: status code %d", rsp.StatusCode)
	}
	var created proto.Instance
	if err := json.NewDecoder(rsp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode instance: %w", err)
	}
	return &created, nil
}

func (c *Client) DeleteInstance(ctx context.Context, id string) error {
	r, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("http://localhost/v1/instances/%s", id), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	rsp, err := c.h.Do(r)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete instance: status code %d", rsp.StatusCode)
	}
	return nil
}

func jsonBody(v any) *bytes.Buffer {
	b := new(bytes.Buffer)
	m, _ := json.Marshal(v)
	b.Write(m)
	return b
}
