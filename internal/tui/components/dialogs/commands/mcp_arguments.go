package commands

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/charmbracelet/crush/internal/llm/agent"
	"github.com/charmbracelet/crush/internal/tui/components/chat"
	"github.com/charmbracelet/crush/internal/tui/components/dialogs"
	"github.com/charmbracelet/crush/internal/tui/styles"
	"github.com/charmbracelet/crush/internal/tui/util"
)

const mcpArgumentsDialogID dialogs.DialogID = "mcp_arguments"

type MCPPromptArgumentsDialog interface {
	dialogs.DialogModel
}

type mcpPromptArgumentsDialogCmp struct {
	wWidth, wHeight int
	width, height   int
	selected        int
	inputs          []textinput.Model
	keys            ArgumentsDialogKeyMap
	id              string
	prompt          *mcp.Prompt
	help            help.Model
}

func NewMCPPromptArgumentsDialog(id, name string) MCPPromptArgumentsDialog {
	id = strings.TrimPrefix(id, MCPPromptPrefix)
	prompt, ok := agent.GetMCPPrompt(id)
	if !ok {
		return nil
	}

	t := styles.CurrentTheme()
	inputs := make([]textinput.Model, len(prompt.Arguments))

	for i, arg := range prompt.Arguments {
		ti := textinput.New()
		placeholder := fmt.Sprintf("Enter value for %s...", arg.Name)
		if arg.Description != "" {
			placeholder = arg.Description
		}
		ti.Placeholder = placeholder
		ti.SetWidth(40)
		ti.SetVirtualCursor(false)
		ti.Prompt = ""
		ti.SetStyles(t.S().TextInput)

		if i == 0 {
			ti.Focus()
		} else {
			ti.Blur()
		}

		inputs[i] = ti
	}

	return &mcpPromptArgumentsDialogCmp{
		inputs: inputs,
		keys:   DefaultArgumentsDialogKeyMap(),
		id:     id,
		prompt: prompt,
		help:   help.New(),
	}
}

func (c *mcpPromptArgumentsDialogCmp) Init() tea.Cmd {
	return nil
}

func (c *mcpPromptArgumentsDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.wWidth = msg.Width
		c.wHeight = msg.Height
		cmd := c.SetSize()
		return c, cmd
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, c.keys.Cancel):
			return c, util.CmdHandler(dialogs.CloseDialogMsg{})
		case key.Matches(msg, c.keys.Confirm):
			if c.selected == len(c.inputs)-1 {
				args := make(map[string]string)
				for i, arg := range c.prompt.Arguments {
					value := c.inputs[i].Value()
					args[arg.Name] = value
				}
				return c, tea.Sequence(
					util.CmdHandler(dialogs.CloseDialogMsg{}),
					c.executeMCPPrompt(args),
				)
			}
			c.inputs[c.selected].Blur()
			c.selected++
			c.inputs[c.selected].Focus()
		case key.Matches(msg, c.keys.Next):
			c.inputs[c.selected].Blur()
			c.selected = (c.selected + 1) % len(c.inputs)
			c.inputs[c.selected].Focus()
		case key.Matches(msg, c.keys.Previous):
			c.inputs[c.selected].Blur()
			c.selected = (c.selected - 1 + len(c.inputs)) % len(c.inputs)
			c.inputs[c.selected].Focus()
		default:
			var cmd tea.Cmd
			c.inputs[c.selected], cmd = c.inputs[c.selected].Update(msg)
			return c, cmd
		}
	}
	return c, nil
}

func (c *mcpPromptArgumentsDialogCmp) executeMCPPrompt(args map[string]string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.SplitN(c.id, ":", 2)
		if len(parts) != 2 {
			return util.ReportError(fmt.Errorf("invalid prompt ID: %s", c.id))
		}
		clientName := parts[0]

		ctx := context.Background()
		slog.Warn("AQUI", "name", c.prompt.Name, "id", c.id)
		result, err := agent.GetMCPPromptContent(ctx, clientName, c.prompt.Name, args)
		if err != nil {
			return util.ReportError(err)
		}

		var content strings.Builder
		for _, msg := range result.Messages {
			if msg.Role == "user" {
				if textContent, ok := msg.Content.(*mcp.TextContent); ok {
					content.WriteString(textContent.Text)
					content.WriteString("\n")
				}
			}
		}

		return chat.SendMsg{
			Text: content.String(),
		}
	}
}

func (c *mcpPromptArgumentsDialogCmp) View() string {
	t := styles.CurrentTheme()
	baseStyle := t.S().Base

	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1).
		Render(cmp.Or(c.prompt.Title, c.prompt.Name))

	promptName := t.S().Text.
		Padding(0, 1).
		Render(c.prompt.Description)

	if c.prompt == nil {
		return baseStyle.Padding(1, 1, 0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus).
			Width(c.width).
			Render(lipgloss.JoinVertical(lipgloss.Left, title, promptName, "", "Prompt not found"))
	}

	inputFields := make([]string, len(c.inputs))
	for i, input := range c.inputs {
		labelStyle := baseStyle.Padding(1, 1, 0, 1)

		if i == c.selected {
			labelStyle = labelStyle.Foreground(t.FgBase).Bold(true)
		} else {
			labelStyle = labelStyle.Foreground(t.FgMuted)
		}

		argName := c.prompt.Arguments[i].Name
		if c.prompt.Arguments[i].Required {
			argName += " *"
		}
		label := labelStyle.Render(argName + ":")

		field := t.S().Text.
			Padding(0, 1).
			Render(input.View())

		inputFields[i] = lipgloss.JoinVertical(lipgloss.Left, label, field)
	}

	elements := []string{title, promptName}
	elements = append(elements, inputFields...)

	c.help.ShowAll = false
	helpText := baseStyle.Padding(0, 1).Render(c.help.View(c.keys))
	elements = append(elements, "", helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, elements...)

	return baseStyle.Padding(1, 1, 0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Width(c.width).
		Render(content)
}

func (c *mcpPromptArgumentsDialogCmp) Cursor() *tea.Cursor {
	if len(c.inputs) == 0 {
		return nil
	}
	cursor := c.inputs[c.selected].Cursor()
	if cursor != nil {
		cursor = c.moveCursor(cursor)
	}
	return cursor
}

const (
	headerHeight      = 3
	itemHeight        = 3
	paddingHorizontal = 3
)

func (c *mcpPromptArgumentsDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := c.Position()
	offset := row + headerHeight + (1+c.selected)*itemHeight
	cursor.Y += offset
	cursor.X = cursor.X + col + paddingHorizontal
	return cursor
}

func (c *mcpPromptArgumentsDialogCmp) SetSize() tea.Cmd {
	c.width = min(90, c.wWidth)
	c.height = min(15, c.wHeight)
	for i := range c.inputs {
		c.inputs[i].SetWidth(c.width - (paddingHorizontal * 2))
	}
	return nil
}

func (c *mcpPromptArgumentsDialogCmp) Position() (int, int) {
	row := (c.wHeight / 2) - (c.height / 2)
	col := (c.wWidth / 2) - (c.width / 2)
	return row, col
}

func (c *mcpPromptArgumentsDialogCmp) ID() dialogs.DialogID {
	return mcpArgumentsDialogID
}
