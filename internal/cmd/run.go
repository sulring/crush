package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/crush/internal/server"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [prompt...]",
	Short: "Run a single non-interactive prompt",
	Long: `Run a single prompt in non-interactive mode and exit.
The prompt can be provided as arguments or piped from stdin.`,
	Example: `
# Run a simple prompt
crush run Explain the use of context in Go

# Pipe input from stdin
echo "What is this code doing?" | crush run

# Run with quiet mode (no spinner)
crush run -q "Generate a README for this project"
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		quiet, _ := cmd.Flags().GetBool("quiet")
		hostURL, err := server.ParseHostURL(clientHost)
		if err != nil {
			return fmt.Errorf("invalid host URL: %v", err)
		}

		c, ins, err := setupApp(cmd, hostURL)
		if err != nil {
			return err
		}
		defer func() { c.DeleteInstance(cmd.Context(), ins.ID) }()

		cfg, err := c.GetConfig(cmd.Context(), ins.ID)
		if err != nil {
			return fmt.Errorf("failed to get config: %v", err)
		}

		if !cfg.IsConfigured() {
			return fmt.Errorf("no providers configured - please run 'crush' to set up a provider interactively")
		}

		prompt := strings.Join(args, " ")

		prompt, err = MaybePrependStdin(prompt)
		if err != nil {
			slog.Error("Failed to read from stdin", "error", err)
			return err
		}

		if prompt == "" {
			return fmt.Errorf("no prompt provided")
		}

		// Run non-interactive flow using the App method
		// return c.RunNonInteractive(cmd.Context(), prompt, quiet)
		// TODO: implement non-interactive run
		_ = quiet
		return nil
	},
}

func init() {
	runCmd.Flags().BoolP("quiet", "q", false, "Hide spinner")
}
