package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/client"
	"github.com/charmbracelet/crush/internal/log"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/server"
	"github.com/charmbracelet/crush/internal/tui"
	"github.com/charmbracelet/crush/internal/version"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

var clientHost string

func init() {
	rootCmd.PersistentFlags().StringP("cwd", "c", "", "Current working directory")
	rootCmd.PersistentFlags().StringP("data-dir", "D", "", "Custom crush data directory")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug")

	rootCmd.Flags().BoolP("help", "h", false, "Help")
	rootCmd.Flags().BoolP("yolo", "y", false, "Automatically accept all permissions (dangerous mode)")

	rootCmd.Flags().StringVar(&clientHost, "host", server.DefaultAddr(), "Connect to a specific crush server host (for advanced users)")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(updateProvidersCmd)
}

var rootCmd = &cobra.Command{
	Use:   "crush",
	Short: "Terminal-based AI assistant for software development",
	Long: `Crush is a powerful terminal-based AI assistant that helps with software development tasks.
It provides an interactive chat interface with AI capabilities, code analysis, and LSP integration
to assist developers in writing, debugging, and understanding code directly from the terminal.`,
	Example: `
# Run in interactive mode
crush

# Run with debug logging
crush -d

# Run with debug logging in a specific directory
crush -d -c /path/to/project

# Run with custom data directory
crush -D /path/to/custom/.crush

# Print version
crush -v

# Run a single non-interactive prompt
crush run "Explain the use of context in Go"

# Run in dangerous mode (auto-accept all permissions)
crush -y
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := setupApp(cmd)
		if err != nil {
			return err
		}

		m, err := tui.New(c)
		if err != nil {
			return fmt.Errorf("failed to create TUI model: %v", err)
		}

		defer func() { c.DeleteInstance(cmd.Context(), c.ID()) }()

		// Set up the TUI.
		program := tea.NewProgram(
			m,
			tea.WithAltScreen(),
			tea.WithContext(cmd.Context()),
			tea.WithMouseCellMotion(),            // Use cell motion instead of all motion to reduce event flooding
			tea.WithFilter(tui.MouseEventFilter), // Filter mouse events based on focus state
		)

		evc, err := c.SubscribeEvents(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to subscribe to events: %v", err)
		}

		go streamEvents(cmd.Context(), evc, program)

		if _, err := program.Run(); err != nil {
			slog.Error("TUI run error", "error", err)
			return fmt.Errorf("TUI error: %v", err)
		}
		return nil
	},
}

func Execute() {
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version.Version),
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}

func streamEvents(ctx context.Context, evc <-chan any, p *tea.Program) {
	defer log.RecoverPanic("app.Subscribe", func() {
		slog.Info("TUI subscription panic: attempting graceful shutdown")
		p.Quit()
	})

	for {
		select {
		case <-ctx.Done():
			slog.Debug("TUI message handler shutting down")
			return
		case ev, ok := <-evc:
			if !ok {
				slog.Debug("TUI message channel closed")
				return
			}
			p.Send(ev)
		}
	}
}

// setupApp handles the common setup logic for both interactive and non-interactive modes.
// It returns the app instance, config, cleanup function, and any error.
func setupApp(cmd *cobra.Command) (*client.Client, error) {
	debug, _ := cmd.Flags().GetBool("debug")
	yolo, _ := cmd.Flags().GetBool("yolo")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	ctx := cmd.Context()

	cwd, err := ResolveCwd(cmd)
	if err != nil {
		return nil, err
	}

	c, err := client.NewClient(cwd, "unix", clientHost)
	if err != nil {
		return nil, err
	}

	if _, err := c.CreateInstance(ctx, proto.Instance{
		Path:    cwd,
		DataDir: dataDir,
		Debug:   debug,
		YOLO:    yolo,
	}); err != nil {
		return nil, fmt.Errorf("failed to create or connect to instance: %v", err)
	}

	return c, nil
}

func MaybePrependStdin(prompt string) (string, error) {
	if term.IsTerminal(os.Stdin.Fd()) {
		return prompt, nil
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return prompt, err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return prompt, nil
	}
	bts, err := io.ReadAll(os.Stdin)
	if err != nil {
		return prompt, err
	}
	return string(bts) + "\n\n" + prompt, nil
}

func ResolveCwd(cmd *cobra.Command) (string, error) {
	cwd, _ := cmd.Flags().GetString("cwd")
	if cwd != "" {
		err := os.Chdir(cwd)
		if err != nil {
			return "", fmt.Errorf("failed to change directory: %v", err)
		}
		return cwd, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}
	return cwd, nil
}

func createDotCrushDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create data directory: %q %w", dir, err)
	}

	gitIgnorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitIgnorePath, []byte("*\n"), 0o644); err != nil {
			return fmt.Errorf("failed to create .gitignore file: %q %w", gitIgnorePath, err)
		}
	}

	return nil
}
