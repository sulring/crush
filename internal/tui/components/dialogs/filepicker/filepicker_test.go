package filepicker

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pngMagicNumberData = []byte("\x89PNG\x0D\x0A\x1A\x0A")

func TestOnPasteMockFSWithValidPath(t *testing.T) {
	mockFS := fstest.MapFS{
		"image1.png": &fstest.MapFile{
			Data: pngMagicNumberData,
		},
		"image2.png": &fstest.MapFile{
			Data: []byte("fake png content"),
		},
	}

	// Test with the first file
	cmd := onPaste(mockFS, "image1.png")
	msg := cmd()

	filePickedMsg, ok := msg.(FilePickedMsg)
	require.True(t, ok)
	require.NotNil(t, filePickedMsg)
	assert.Equal(t, "image1.png", filePickedMsg.Attachment.FileName)
	assert.Equal(t, "image/png", filePickedMsg.Attachment.MimeType)
}
