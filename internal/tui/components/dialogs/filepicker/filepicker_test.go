package filepicker

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pngMagicNumberData = []byte("\x89PNG\x0D\x0A\x1A\x0A")

func TestOnPasteMockFSWithValidPath(t *testing.T) {
	var mockedFSPath string
	resolveTestFS := func(fsysPath string) fs.FS {
		mockedFSPath = fsysPath
		return fstest.MapFS{
			"image1.png": &fstest.MapFile{
				Data: pngMagicNumberData,
			},
			"image2.png": &fstest.MapFile{
				Data: []byte("fake png content"),
			},
		}
	}

	// Test with the first file
	cmd := onPaste(resolveTestFS, "/home/testuser/images/image1.png")
	msg := cmd()

	assert.Equal(t, "/home/testuser/images", mockedFSPath)
	filePickedMsg, ok := msg.(FilePickedMsg)
	require.True(t, ok)
	require.NotNil(t, filePickedMsg)
	assert.Equal(t, "image1.png", filePickedMsg.Attachment.FileName)
	assert.Equal(t, "image/png", filePickedMsg.Attachment.MimeType)
}
