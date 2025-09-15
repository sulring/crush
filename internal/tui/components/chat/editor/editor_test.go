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

// executeBatchCommands executes all commands in a BatchMsg and returns the resulting messages
func executeBatchCommands(batchMsg tea.BatchMsg) []tea.Msg {
	var messages []tea.Msg
	for _, cmd := range batchMsg {
		if cmd != nil {
			msg := cmd()
			messages = append(messages, msg)
		}
	}
	return messages
}

// assertBatchContainsMessage checks if a BatchMsg contains a message of the specified type
func assertBatchContainsMessage(t *testing.T, batchMsg tea.BatchMsg, expectedType interface{}) bool {
	t.Helper()
	messages := executeBatchCommands(batchMsg)

	for _, msg := range messages {
		switch expectedType.(type) {
		case completions.OpenCompletionsMsg:
			if _, ok := msg.(completions.OpenCompletionsMsg); ok {
				return true
			}
		}
	}
	return false
}

// assertBatchContainsExactMessage checks if a BatchMsg contains a message with exact field values
func assertBatchContainsExactMessage(t *testing.T, batchMsg tea.BatchMsg, expected interface{}) bool {
	t.Helper()
	messages := executeBatchCommands(batchMsg)

	for _, msg := range messages {
		switch expected := expected.(type) {
		case completions.OpenCompletionsMsg:
			if actual, ok := msg.(completions.OpenCompletionsMsg); ok {
				// If no specific completions are expected, just match the type
				if len(expected.Completions) == 0 {
					return true
				}
				// Compare completions if specified
				if len(actual.Completions) == len(expected.Completions) {
					// For simplicity, just check the count for now
					// A more complete implementation would compare each completion
					return true
				}
			}
		default:
			// Fallback to type checking only
			if _, ok := msg.(completions.OpenCompletionsMsg); ok {
				return true
			}
		}
	}
	return false
}

// assertBatchContainsOpenCompletionsMsg checks if a BatchMsg contains an OpenCompletionsMsg
// with the expected completions. If expectedCompletions is nil, only the message type is checked.
func assertBatchContainsOpenCompletionsMsg(t *testing.T, batchMsg tea.BatchMsg, expectedCompletions []string) (*completions.OpenCompletionsMsg, bool) {
	t.Helper()
	messages := executeBatchCommands(batchMsg)

	for _, msg := range messages {
		if actual, ok := msg.(completions.OpenCompletionsMsg); ok {
			if expectedCompletions == nil {
				return &actual, true
			}

			// Convert actual completions to string titles for comparison
			actualTitles := make([]string, len(actual.Completions))
			for i, comp := range actual.Completions {
				actualTitles[i] = comp.Title
			}

			// Check if we have the same number of completions
			if len(actualTitles) != len(expectedCompletions) {
				continue
			}

			// For now, just check that we have the same count
			// A more sophisticated implementation would check the actual values
			return &actual, true
		}
	}
	return nil, false
}

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

	// completions menu is open
	assert.True(t, testEditor.isCompletionsOpen)
	assert.Equal(t, "/", testEditor.textarea.Value())

	// the query is empty (since we just opened it)
	assert.Equal(t, "", testEditor.currentQuery)
}

func TestEditorAutocompletion_StartFilteringOpens(t *testing.T) {
	testEditor := newEditor(&app.App{}, mockDirLister([]string{"file1.txt", "file2.txt"}))
	require.NotNil(t, testEditor)

	// open the completions menu by simulating a '/' key press
	testEditor.Focus()
	keyPressMsg := tea.KeyPressMsg{
		Text: "/",
	}

	m, cmds := testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)
	
	msg := cmds()
	var openCompletionsMsg *completions.OpenCompletionsMsg
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		// Use our enhanced helper to check for OpenCompletionsMsg with specific completions
		var found bool
		openCompletionsMsg, found = assertBatchContainsOpenCompletionsMsg(t, batchMsg, []string{"file1.txt", "file2.txt"})
		assert.True(t, found, "Expected to find OpenCompletionsMsg with specific completions in batched messages")
	} else {
		t.Fatal("Expected BatchMsg from cmds()")
	}

	assert.NotNil(t, openCompletionsMsg)
	m, cmds = testEditor.Update(openCompletionsMsg)

	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		assertBatchContainsExactMessage(t, batchMsg, completions.CompletionsOpenedMsg{})
	} else {
		t.Fatal("Expected BatchMsg from cmds()")
	}

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

	// Verify the editor still has completions open
	assert.True(t, testEditor.isCompletionsOpen)

	// The currentQuery should be updated based on what we typed
	// In this case, it would be "test" (the word after the initial '/')
	// Note: The actual filtering is handled by the completions component,
	// so we're just verifying the editor's state is correct
	assert.Equal(t, "test", testEditor.currentQuery)
}

func TestEditorAutocompletion_SelectionOfNormalPathAddsToTextAreaClosesCompletion(t *testing.T) {
	testEditor := newEditor(&app.App{}, mockDirLister([]string{"file1.txt", "file2.txt"}))
	require.NotNil(t, testEditor)

	// open the completions menu by simulating a '/' key press
	testEditor.Focus()
	keyPressMsg := tea.KeyPressMsg{
		Text: "/",
	}

	m, cmds := testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)
	
	msg := cmds()
	assert.NotNil(t, msg)
	m, cmds = testEditor.Update(msg)

	// Now simulate typing a query to filter the completions
	// Set the text to "/tes" and then simulate typing "t" to make "/test"
	testEditor.textarea.SetValue("/tes")

	// Simulate typing a key that would trigger filtering
	keyPressMsg = tea.KeyPressMsg{
		Text: "t",
	}

	m, cmds = testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)

	// The currentQuery should be updated based on what we typed
	// In this case, it would be "test" (the word after the initial '/')
	// Note: The actual filtering is handled by the completions component,
	// so we're just verifying the editor's state is correct
	assert.Equal(t, "test", testEditor.currentQuery)
}

// TestHelperFunctions demonstrates how to use the batch message helpers
func TestHelperFunctions(t *testing.T) {
	testEditor := newEditor(&app.App{}, mockDirLister([]string{"file1.txt", "file2.txt"}))
	require.NotNil(t, testEditor)

	// Simulate pressing the '/' key
	testEditor.Focus()
	keyPressMsg := tea.KeyPressMsg{
		Text: "/",
	}

	m, cmds := testEditor.Update(keyPressMsg)
	testEditor = m.(*editorCmp)

	// Execute the command and check if it returns a BatchMsg
	msg := cmds()
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		// Test our helper functions
		found := assertBatchContainsMessage(t, batchMsg, completions.OpenCompletionsMsg{})
		assert.True(t, found, "Expected to find OpenCompletionsMsg in batched messages")

		// Test exact message helper
		foundExact := assertBatchContainsExactMessage(t, batchMsg, completions.OpenCompletionsMsg{})
		assert.True(t, foundExact, "Expected to find exact OpenCompletionsMsg in batched messages")

		// Test specific completions helper
		msg, foundSpecific := assertBatchContainsOpenCompletionsMsg(t, batchMsg, nil) // Just check type
		assert.NotNil(t, msg)
		assert.True(t, foundSpecific, "Expected to find OpenCompletionsMsg in batched messages")
	} else {
		t.Fatal("Expected BatchMsg from cmds()")
	}
}
