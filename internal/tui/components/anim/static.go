package anim

import (
	"image/color"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type noAnim struct {
	Color    color.Color
	rendered string
}

func newStatic(label string, foreground color.Color) Spinner {
	a := &noAnim{Color: foreground}
	a.SetLabel(label)
	return a
}

func (s *noAnim) SetLabel(label string) {
	s.rendered = lipgloss.NewStyle().Foreground(s.Color).Render(label + ellipsisFrames[2])
}

func (s noAnim) Init() tea.Cmd                      { return nil }
func (s *noAnim) Update(tea.Msg) (Spinner, tea.Cmd) { return s, nil }
func (s *noAnim) View() string                      { return s.rendered }
