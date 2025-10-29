package tools

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/shell"
)

const (
	BashOutputToolName = "bash_output"
)

type BashOutputParams struct {
	ShellID string `json:"shell_id" description:"The ID of the background shell to retrieve output from"`
}

type BashOutputResponseMetadata struct {
	ShellID          string `json:"shell_id"`
	Done             bool   `json:"done"`
	WorkingDirectory string `json:"working_directory"`
}

const bashOutputDescription = `Retrieves the current output from a background shell.

<usage>
- Provide the shell ID returned from a background bash execution
- Returns the current stdout and stderr output
- Indicates whether the shell has completed execution
</usage>

<features>
- View output from running background processes
- Check if background process has completed
- Get cumulative output from process start
</features>

<tips>
- Use this to monitor long-running processes
- Check the 'done' status to see if process completed
- Can be called multiple times to view incremental output
</tips>
`

func NewBashOutputTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		BashOutputToolName,
		bashOutputDescription,
		func(ctx context.Context, params BashOutputParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.ShellID == "" {
				return fantasy.NewTextErrorResponse("missing shell_id"), nil
			}

			bgManager := shell.GetBackgroundShellManager()
			bgShell, ok := bgManager.Get(params.ShellID)
			if !ok {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("background shell not found: %s", params.ShellID)), nil
			}

			stdout, stderr, done, err := bgShell.GetOutput()

			var outputParts []string
			if stdout != "" {
				outputParts = append(outputParts, stdout)
			}
			if stderr != "" {
				outputParts = append(outputParts, stderr)
			}

			status := "running"
			if done {
				status = "completed"
				if err != nil {
					exitCode := shell.ExitCode(err)
					if exitCode != 0 {
						outputParts = append(outputParts, fmt.Sprintf("Exit code %d", exitCode))
					}
				}
			}

			output := strings.Join(outputParts, "\n")

			metadata := BashOutputResponseMetadata{
				ShellID:          params.ShellID,
				Done:             done,
				WorkingDirectory: bgShell.GetWorkingDir(),
			}

			if output == "" {
				output = BashNoOutput
			}

			result := fmt.Sprintf("Shell ID: %s\nStatus: %s\n\nOutput:\n%s", params.ShellID, status, output)
			return fantasy.WithResponseMetadata(fantasy.NewTextResponse(result), metadata), nil
		})
}
