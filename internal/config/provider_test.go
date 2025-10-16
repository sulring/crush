package config

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

type mockProviderClient struct {
	shouldFail bool
}

func (m *mockProviderClient) GetProviders() ([]catwalk.Provider, error) {
	if m.shouldFail {
		return nil, errors.New("failed to load providers")
	}
	return []catwalk.Provider{
		{
			Name: "Mock",
		},
	}, nil
}

func TestProvider_loadProvidersNoIssues(t *testing.T) {
	client := &mockProviderClient{shouldFail: false}
	tmpPath := t.TempDir() + "/providers.json"
	cfg := &Config{}
	providers, err := loadProviders(false, client, tmpPath, cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	require.Len(t, providers, 1)

	// check if file got saved
	fileInfo, err := os.Stat(tmpPath)
	require.NoError(t, err)
	require.False(t, fileInfo.IsDir(), "Expected a file, not a directory")
}

func TestProvider_loadProvidersWithIssues(t *testing.T) {
	client := &mockProviderClient{shouldFail: true}
	tmpPath := t.TempDir() + "/providers.json"
	// store providers to a temporary file
	oldProviders := []catwalk.Provider{
		{
			Name: "OldProvider",
		},
	}
	data, err := json.Marshal(oldProviders)
	if err != nil {
		t.Fatalf("Failed to marshal old providers: %v", err)
	}

	err = os.WriteFile(tmpPath, data, 0o644)
	if err != nil {
		t.Fatalf("Failed to write old providers to file: %v", err)
	}
	cfg := &Config{}
	providers, err := loadProviders(false, client, tmpPath, cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	require.Len(t, providers, 1)
	require.Equal(t, "OldProvider", providers[0].Name, "Expected to keep old provider when loading fails")
}

func TestProvider_loadProvidersWithIssuesAndNoCache(t *testing.T) {
	client := &mockProviderClient{shouldFail: true}
	tmpPath := t.TempDir() + "/providers.json"
	cfg := &Config{}
	providers, err := loadProviders(false, client, tmpPath, cfg)
	require.Error(t, err)
	require.Nil(t, providers, "Expected nil providers when loading fails and no cache exists")
}

type dynamicMockProviderClient struct {
	mu        sync.Mutex
	callCount int
	providers [][]catwalk.Provider
}

func (m *dynamicMockProviderClient) GetProviders() ([]catwalk.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.callCount >= len(m.providers) {
		return m.providers[len(m.providers)-1], nil
	}

	result := m.providers[m.callCount]
	m.callCount++
	return result, nil
}

func TestProvider_backgroundReloadAfterCacheUpdate(t *testing.T) {
	t.Parallel()

	tmpPath := t.TempDir() + "/providers.json"

	initialProviders := []catwalk.Provider{
		{Name: "InitialProvider"},
	}
	updatedProviders := []catwalk.Provider{
		{Name: "UpdatedProvider"},
		{Name: "NewProvider"},
	}

	client := &dynamicMockProviderClient{
		providers: [][]catwalk.Provider{
			updatedProviders,
		},
	}

	data, err := json.Marshal(initialProviders)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tmpPath, data, 0o644))

	providerMu.Lock()
	oldInitialized := initialized
	initialized = false
	providerMu.Unlock()

	defer func() {
		providerMu.Lock()
		initialized = oldInitialized
		providerMu.Unlock()
	}()

	cfg := &Config{}
	providers, err := loadProviders(false, client, tmpPath, cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	require.Len(t, providers, 1)
	require.Equal(t, "InitialProvider", providers[0].Name)

	require.Eventually(t, func() bool {
		reloadedProviders, err := loadProvidersFromCache(tmpPath)
		if err != nil {
			return false
		}
		return len(reloadedProviders) == 2
	}, 2*time.Second, 50*time.Millisecond, "Background cache update should complete within 2 seconds")

	reloadedProviders, err := loadProvidersFromCache(tmpPath)
	require.NoError(t, err)
	require.Len(t, reloadedProviders, 2)
	require.Equal(t, "UpdatedProvider", reloadedProviders[0].Name)
	require.Equal(t, "NewProvider", reloadedProviders[1].Name)

	require.Eventually(t, func() bool {
		providerMu.RLock()
		defer providerMu.RUnlock()
		return len(providerList) == 2
	}, 2*time.Second, 50*time.Millisecond, "In-memory provider list should be reloaded")

	providerMu.RLock()
	inMemoryProviders := providerList
	providerMu.RUnlock()

	require.Len(t, inMemoryProviders, 2)
	require.Equal(t, "UpdatedProvider", inMemoryProviders[0].Name)
	require.Equal(t, "NewProvider", inMemoryProviders[1].Name)
}

func TestProvider_reloadProvidersThreadSafety(t *testing.T) {
	tmpPath := t.TempDir() + "/providers.json"

	initialProviders := []catwalk.Provider{
		{Name: "Provider1"},
	}
	data, err := json.Marshal(initialProviders)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tmpPath, data, 0o644))

	providerMu.Lock()
	oldList := providerList
	oldErr := providerErr
	oldInitialized := initialized
	providerList = initialProviders
	providerErr = nil
	initialized = true
	providerMu.Unlock()

	defer func() {
		providerMu.Lock()
		providerList = oldList
		providerErr = oldErr
		initialized = oldInitialized
		providerMu.Unlock()
	}()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()

			updatedProviders := []catwalk.Provider{
				{Name: "Provider1"},
				{Name: "Provider2"},
			}
			data, err := json.Marshal(updatedProviders)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(tmpPath, data, 0o644))

			reloadProviders(tmpPath)

			providerMu.RLock()
			currentList := providerList
			providerMu.RUnlock()

			require.NotNil(t, currentList)
		}(i)
	}

	wg.Wait()

	providerMu.RLock()
	finalList := providerList
	providerMu.RUnlock()

	require.Len(t, finalList, 2)
}

func TestProvider_reloadProvidersWithEmptyCache(t *testing.T) {
	tmpPath := t.TempDir() + "/providers.json"

	initialProviders := []catwalk.Provider{
		{Name: "InitialProvider"},
	}

	providerMu.Lock()
	oldList := providerList
	oldErr := providerErr
	oldInitialized := initialized
	providerList = initialProviders
	providerErr = nil
	initialized = true
	providerMu.Unlock()

	defer func() {
		providerMu.Lock()
		providerList = oldList
		providerErr = oldErr
		initialized = oldInitialized
		providerMu.Unlock()
	}()

	emptyProviders := []catwalk.Provider{}
	data, err := json.Marshal(emptyProviders)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tmpPath, data, 0o644))

	reloadProviders(tmpPath)

	providerMu.RLock()
	currentList := providerList
	providerMu.RUnlock()

	require.Len(t, currentList, 1)
	require.Equal(t, "InitialProvider", currentList[0].Name)
}

func TestProvider_reloadProvidersWithInvalidCache(t *testing.T) {
	tmpPath := t.TempDir() + "/providers.json"

	initialProviders := []catwalk.Provider{
		{Name: "InitialProvider"},
	}

	providerMu.Lock()
	oldList := providerList
	oldErr := providerErr
	oldInitialized := initialized
	providerList = initialProviders
	providerErr = nil
	initialized = true
	providerMu.Unlock()

	defer func() {
		providerMu.Lock()
		providerList = oldList
		providerErr = oldErr
		initialized = oldInitialized
		providerMu.Unlock()
	}()

	require.NoError(t, os.WriteFile(tmpPath, []byte("invalid json"), 0o644))

	reloadProviders(tmpPath)

	providerMu.RLock()
	currentList := providerList
	providerMu.RUnlock()

	require.Len(t, currentList, 1)
	require.Equal(t, "InitialProvider", currentList[0].Name)
}
