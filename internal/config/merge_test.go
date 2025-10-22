package config

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfigMerging defines the rules on how configuration merging works.
// Generally, things are either appended to or replaced by the later configuration.
// Whether one or the other happen depends on effects its effects.
func TestConfigMerging(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		c := exerciseMerge(t, Config{}, Config{})
		require.NotNil(t, c)
	})

	t.Run("mcps", func(t *testing.T) {
		c := exerciseMerge(t, Config{
			MCP: MCPs{
				"foo": {
					Command: "foo-mcp",
					Args:    []string{"serve"},
					Type:    MCPSSE,
					Timeout: 10,
				},
				"zaz": {
					Disabled: true,
					Env:      map[string]string{"FOO": "bar"},
					Headers:  map[string]string{"api-key": "exposed"},
					URL:      "nope",
				},
			},
		}, Config{
			MCP: MCPs{
				"foo": {
					Args:    []string{"serve", "--stdio"},
					Type:    MCPStdio,
					Timeout: 7,
				},
				"bar": {
					Command: "bar",
				},
				"zaz": {
					Env:     map[string]string{"FOO": "foo", "BAR": "bar"},
					Headers: map[string]string{"api-key": "$API"},
					URL:     "http://bar",
				},
			},
		})
		require.NotNil(t, c)
		require.Len(t, slices.Collect(maps.Keys(c.MCP)), 3)
		require.Equal(t, MCPConfig{
			Command: "foo-mcp",
			Args:    []string{"serve", "--stdio"},
			Type:    MCPStdio,
			Timeout: 10,
		}, c.MCP["foo"])
		require.Equal(t, MCPConfig{
			Command: "bar",
		}, c.MCP["bar"])
		require.Equal(t, MCPConfig{
			Disabled: true,
			URL:      "http://bar",
			Env:      map[string]string{"FOO": "foo", "BAR": "bar"},
			Headers:  map[string]string{"api-key": "$API"},
		}, c.MCP["zaz"])
	})

	t.Run("lsps", func(t *testing.T) {
		result := exerciseMerge(t, Config{
			LSP: LSPs{
				"gopls": LSPConfig{
					Env:         map[string]string{"FOO": "bar"},
					RootMarkers: []string{"go.sum"},
					FileTypes:   []string{"go"},
				},
			},
		}, Config{
			LSP: LSPs{
				"gopls": LSPConfig{
					Command:     "gopls",
					InitOptions: map[string]any{"a": 10},
					RootMarkers: []string{"go.sum"},
				},
			},
		}, Config{
			LSP: LSPs{
				"gopls": LSPConfig{
					Args:        []string{"serve", "--stdio"},
					InitOptions: map[string]any{"a": 12, "b": 18},
					RootMarkers: []string{"go.sum", "go.mod"},
					FileTypes:   []string{"go"},
					Disabled:    true,
				},
			},
		},
			Config{
				LSP: LSPs{
					"gopls": LSPConfig{
						Options:     map[string]any{"opt1": "10"},
						RootMarkers: []string{"go.work"},
					},
				},
			},
		)
		require.NotNil(t, result)
		require.Equal(t, LSPConfig{
			Disabled:    true,
			Command:     "gopls",
			Args:        []string{"serve", "--stdio"},
			Env:         map[string]string{"FOO": "bar"},
			FileTypes:   []string{"go"},
			RootMarkers: []string{"go.mod", "go.sum", "go.work"},
			InitOptions: map[string]any{"a": 12.0, "b": 18.0},
			Options:     map[string]any{"opt1": "10"},
		}, result.LSP["gopls"])
	})
}

func exerciseMerge(tb testing.TB, confs ...Config) *Config {
	tb.Helper()
	readers := make([]io.Reader, 0, len(confs))
	for _, c := range confs {
		bts, err := json.Marshal(c)
		require.NoError(tb, err)
		readers = append(readers, bytes.NewReader(bts))
	}
	result, err := loadFromReaders(readers)
	require.NoError(tb, err)
	return result
}
