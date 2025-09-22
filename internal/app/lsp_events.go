package app

import (
	"context"
	"maps"
	"time"

	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/pubsub"
)

// LSPEventType represents the type of LSP event
type LSPEventType string

const (
	LSPEventStateChanged       LSPEventType = "state_changed"
	LSPEventDiagnosticsChanged LSPEventType = "diagnostics_changed"
)

func (e LSPEventType) MarshalText() ([]byte, error) {
	return []byte(e), nil
}

func (e *LSPEventType) UnmarshalText(data []byte) error {
	*e = LSPEventType(data)
	return nil
}

// LSPEvent represents an event in the LSP system
type LSPEvent struct {
	Type            LSPEventType    `json:"type"`
	Name            string          `json:"name"`
	State           lsp.ServerState `json:"state"`
	Error           error           `json:"error,omitempty"`
	DiagnosticCount int             `json:"diagnostic_count,omitempty"`
}

// LSPClientInfo holds information about an LSP client's state
type LSPClientInfo struct {
	Name            string          `json:"name"`
	State           lsp.ServerState `json:"state"`
	Error           error           `json:"error,omitempty"`
	DiagnosticCount int             `json:"diagnostic_count,omitempty"`
	ConnectedAt     time.Time       `json:"connected_at"`
}

// SubscribeLSPEvents returns a channel for LSP events
func (a *App) SubscribeLSPEvents(ctx context.Context) <-chan pubsub.Event[LSPEvent] {
	return a.lspBroker.Subscribe(ctx)
}

// GetLSPStates returns the current state of all LSP clients
func (a *App) GetLSPStates() map[string]LSPClientInfo {
	return maps.Collect(a.lspStates.Seq2())
}

// GetLSPState returns the state of a specific LSP client
func (a *App) GetLSPState(name string) (LSPClientInfo, bool) {
	return a.lspStates.Get(name)
}

// updateLSPState updates the state of an LSP client and publishes an event
func (a *App) updateLSPState(name string, state lsp.ServerState, err error, diagnosticCount int) {
	info := LSPClientInfo{
		Name:            name,
		State:           state,
		Error:           err,
		DiagnosticCount: diagnosticCount,
	}
	if state == lsp.StateReady {
		info.ConnectedAt = time.Now()
	}
	a.lspStates.Set(name, info)

	// Publish state change event
	a.lspBroker.Publish(pubsub.UpdatedEvent, LSPEvent{
		Type:            LSPEventStateChanged,
		Name:            name,
		State:           state,
		Error:           err,
		DiagnosticCount: diagnosticCount,
	})
}

// updateLSPDiagnostics updates the diagnostic count for an LSP client and publishes an event
func (a *App) updateLSPDiagnostics(name string, diagnosticCount int) {
	if info, exists := a.lspStates.Get(name); exists {
		info.DiagnosticCount = diagnosticCount
		a.lspStates.Set(name, info)

		// Publish diagnostics change event
		a.lspBroker.Publish(pubsub.UpdatedEvent, LSPEvent{
			Type:            LSPEventDiagnosticsChanged,
			Name:            name,
			State:           info.State,
			Error:           info.Error,
			DiagnosticCount: diagnosticCount,
		})
	}
}
