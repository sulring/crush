package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/fsext"
	"github.com/charmbracelet/crush/internal/tui/components/completions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockDirLister(paths []string) fsext.DirectoryListerResolver {
	return func() fsext.DirectoryLister {
		return func(initialPath string, ignorePatterns []string, limit int) ([]string, bool, error) {
			return paths, false, nil
		}
	}
}

func TestEditorTypingForwardSlashOpensCompletions(t *testing.T) {
	testEditor := newEditor(&app.App{}, mockDirLister([]string{}))
	require.NotNil(t, testEditor)

	// Simulate pressing the '/' key
	keyPressMsg := tea.KeyPressMsg{
		Text: "/",
	}

	m, cmds := testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)
	cmds()

	assert.True(t, testEditor.isCompletionsOpen)
	assert.Equal(t, "/", testEditor.textarea.Value())
}

func TestEditorAutocompletionWithEmptyInput(t *testing.T) {
	testEditor := newEditor(&app.App{}, mockDirLister([]string{}))
	require.NotNil(t, testEditor)

	// First, give the editor focus
	testEditor.Focus()

	// Simulate pressing the '/' key when the editor is empty
	// This should trigger the completions to open
	keyPressMsg := tea.KeyPressMsg{
		Text: "/",
	}

	m, cmds := testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)
	cmds()

	// Verify completions menu is open
	assert.True(t, testEditor.isCompletionsOpen)
	assert.Equal(t, "/", testEditor.textarea.Value())

	// Verify the query is empty (since we just opened it)
	assert.Equal(t, "", testEditor.currentQuery)
}

func TestEditorAutocompletionFiltering(t *testing.T) {
	testEditor := newEditor(&app.App{}, mockDirLister([]string{}))
	require.NotNil(t, testEditor)

	// First, open the completions menu by simulating a '/' key press
	testEditor.Focus()
	keyPressMsg := tea.KeyPressMsg{
		Text: "/",
	}

	m, cmds := testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)
	
	// Execute the command and check if it returns a BatchMsg
	msg := cmds()
	foundOpenCompletions := false
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		// Handle batched messages
		for _, cmd := range batchMsg {
			if cmd != nil {
				resultMsg := cmd()
				if _, ok := resultMsg.(completions.OpenCompletionsMsg); ok {
					foundOpenCompletions = true
					break
				}
			}
		}
	} else {
		t.Fatal("Expected BatchMsg from cmds()")
	}
	assert.True(t, foundOpenCompletions, "Expected to find OpenCompletionsMsg in batched messages")

	// Verify completions menu is open
	assert.True(t, testEditor.isCompletionsOpen)
	assert.Equal(t, "/", testEditor.textarea.Value())

	// Now simulate typing a query to filter the completions
	// Set the text to "/tes" and then simulate typing "t" to make "/test"
	testEditor.textarea.SetValue("/tes")

	// Simulate typing a key that would trigger filtering
	keyPressMsg = tea.KeyPressMsg{
		Text: "t",
	}

	m, cmds = testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)
	cmds()

	// Verify the editor still has completions open
	assert.True(t, testEditor.isCompletionsOpen)

	// The currentQuery should be updated based on what we typed
	// In this case, it would be "test" (the word after the initial '/')
	// Note: The actual filtering is handled by the completions component,
	// so we're just verifying the editor's state is correct
	assert.Equal(t, "test", testEditor.currentQuery)
}
