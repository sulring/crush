package app

import (
	"context"
	"maps"
	"time"

	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
)

type (
	LSPClientInfo = proto.LSPClientInfo
	LSPEvent      = proto.LSPEvent
)

const (
	LSPEventStateChanged       = proto.LSPEventStateChanged
	LSPEventDiagnosticsChanged = proto.LSPEventDiagnosticsChanged
)

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
