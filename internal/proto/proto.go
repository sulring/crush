package proto

import (
	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/message"
)

// Instance represents a running app.App instance with its associated resources
// and state.
type Instance struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	YOLO    bool   `json:"yolo,omitempty"`
	Debug   bool   `json:"debug,omitempty"`
	DataDir string `json:"data_dir,omitempty"`
}

// Error represents an error response.
type Error struct {
	Message string `json:"message"`
}

// AgentInfo represents information about the agent.
type AgentInfo struct {
	IsBusy bool          `json:"is_busy"`
	Model  catwalk.Model `json:"model"`
}

// IsZero checks if the AgentInfo is zero-valued.
func (a AgentInfo) IsZero() bool {
	return a == AgentInfo{}
}

// AgentMessage represents a message sent to the agent.
type AgentMessage struct {
	SessionID   string               `json:"session_id"`
	Prompt      string               `json:"prompt"`
	Attachments []message.Attachment `json:"attachments,omitempty"`
}
