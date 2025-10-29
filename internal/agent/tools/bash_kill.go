package tools

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/shell"
)

const (
	BashKillToolName = "bash_kill"
)

type BashKillParams struct {
	ShellID string `json:"shell_id" description:"The ID of the background shell to terminate"`
}

type BashKillResponseMetadata struct {
	ShellID string `json:"shell_id"`
}

const bashKillDescription = `Terminates a background shell process.

<usage>
- Provide the shell ID returned from a background bash execution
- Cancels the running process and cleans up resources
</usage>

<features>
- Stop long-running background processes
- Clean up completed background shells
- Immediately terminates the process
</features>

<tips>
- Use this when you need to stop a background process
- The process is terminated immediately (similar to SIGTERM)
- After killing, the shell ID becomes invalid
</tips>
`

func NewBashKillTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		BashKillToolName,
		bashKillDescription,
		func(ctx context.Context, params BashKillParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.ShellID == "" {
				return fantasy.NewTextErrorResponse("missing shell_id"), nil
			}

			bgManager := shell.GetBackgroundShellManager()
			err := bgManager.Kill(params.ShellID)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			metadata := BashKillResponseMetadata(params)

			result := fmt.Sprintf("Background shell %s terminated successfully", params.ShellID)
			return fantasy.WithResponseMetadata(fantasy.NewTextResponse(result), metadata), nil
		})
}
