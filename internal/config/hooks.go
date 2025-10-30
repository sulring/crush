package config

// HookEventType represents the lifecycle event when a hook should run.
type HookEventType string

const (
	// PreToolUse runs before tool calls and can block them.
	PreToolUse HookEventType = "PreToolUse"
	// PostToolUse runs after tool calls complete.
	PostToolUse HookEventType = "PostToolUse"
	// UserPromptSubmit runs when the user submits a prompt, before processing.
	UserPromptSubmit HookEventType = "UserPromptSubmit"
	// Notification runs when Crush sends notifications.
	Notification HookEventType = "Notification"
	// Stop runs when Crush finishes responding.
	Stop HookEventType = "Stop"
	// SubagentStop runs when subagent tasks complete.
	SubagentStop HookEventType = "SubagentStop"
	// PreCompact runs before running a compact operation.
	PreCompact HookEventType = "PreCompact"
	// SessionStart runs when a session starts or resumes.
	SessionStart HookEventType = "SessionStart"
	// SessionEnd runs when a session ends.
	SessionEnd HookEventType = "SessionEnd"
)

// Hook represents a single hook command configuration.
type Hook struct {
	// Type is the hook type, currently only "command" is supported.
	Type string `json:"type" jsonschema:"description=Hook type,enum=command,default=command"`
	// Command is the shell command to execute.
	Command string `json:"command" jsonschema:"required,description=Shell command to execute for this hook,example=echo 'Hook executed'"`
	// Timeout is the maximum time in seconds to wait for the hook to complete.
	// Default is 30 seconds.
	Timeout *int `json:"timeout,omitempty" jsonschema:"description=Maximum time in seconds to wait for hook completion,default=30,minimum=1,maximum=300"`
}

// HookMatcher represents a matcher for a specific event type.
type HookMatcher struct {
	// Matcher is the tool name or pattern to match (for tool events).
	// For non-tool events, this can be empty or "*" to match all.
	Matcher string `json:"matcher,omitempty" jsonschema:"description=Tool name or pattern to match (e.g. 'bash' or '*' for all),example=bash,example=edit,example=*"`
	// Hooks is the list of hooks to execute when the matcher matches.
	Hooks []Hook `json:"hooks" jsonschema:"required,description=List of hooks to execute when matcher matches"`
}

// HookConfig holds the complete hook configuration.
type HookConfig map[HookEventType][]HookMatcher
