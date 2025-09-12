package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/assert"
)

func TestEditorTypingForwardSlashOpensCompletions(t *testing.T) {
	testEditor := newEditor(&app.App{})
	require.NotNil(t, testEditor)

	// Simulate pressing the '/' key
	keyPressMsg := tea.KeyPressMsg{
		Text: "/",
	}

	m, cmds := testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)
	cmds()

	assert.True(t, testEditor.isCompletionsOpen)
	assert.Equal(t, testEditor.textarea.Value(), "/")
}
