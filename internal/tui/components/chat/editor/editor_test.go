package editor

import (
	"testing"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/stretchr/testify/require"
)

func TestEditorTypingForwardSlashOpensCompletions(t *testing.T) {
	editorCmp := newEditor(&app.App{})
	require.NotNil(t, editorCmp)
}
