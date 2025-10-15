package update

import (
	"context"
	"testing"

	"github.com/charmbracelet/crush/internal/version"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdate_DevelopmentVersion(t *testing.T) {
	// Test that development versions don't trigger updates.
	ctx := context.Background()

	// Temporarily set version to development version.
	originalVersion := version.Version
	version.Version = "unknown"
	defer func() {
		version.Version = originalVersion
	}()

	info, err := Check(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.False(t, info.Available)
}
