package update

import (
	"context"
	"testing"

	"github.com/charmbracelet/crush/internal/version"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdate_DevelopmentVersion(t *testing.T) {
	originalVersion := version.Version
	version.Version = "unknown"
	t.Cleanup(func() {
		version.Version = originalVersion
	})

	info, err := Check(t.Context(), testClient{})
	require.NoError(t, err)
	require.NotNil(t, info)
	require.False(t, info.Available())
}

func TestCheckForUpdate_Old(t *testing.T) {
	originalVersion := version.Version
	version.Version = "0.10.0"
	t.Cleanup(func() {
		version.Version = originalVersion
	})
	info, err := Check(t.Context(), testClient{})
	require.NoError(t, err)
	require.NotNil(t, info)
	require.True(t, info.Available())
}

type testClient struct{}

// Latest implements Client.
func (t testClient) Latest(ctx context.Context) (*Release, error) {
	return &Release{
		TagName: "v0.11.0",
		HTMLURL: "https://example.org",
	}, nil
}
