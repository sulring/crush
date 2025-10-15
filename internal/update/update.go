package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/crush/internal/version"
)

const (
	githubAPIURL = "https://api.github.com/repos/charmbracelet/crush/releases/latest"
	userAgent    = "crush/1.0"
)

// Release represents a GitHub release.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// UpdateInfo contains information about an available update.
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	Available      bool
}

// CheckForUpdate checks if a new version is available.
func CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	info := &UpdateInfo{
		CurrentVersion: version.Version,
	}

	cv, err := semver.NewVersion(version.Version)
	if err != nil {
		// its devel, unknown, etc
		return info, nil
	}

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	lv, err := semver.NewVersion(release.TagName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse latest version: %w", err)
	}

	info.LatestVersion = strings.TrimPrefix(release.TagName, "v")
	info.ReleaseURL = release.HTMLURL
	info.Available = lv.GreaterThan(cv)

	return info, nil
}

// fetchLatestRelease fetches the latest release information from GitHub.
func fetchLatestRelease(ctx context.Context) (*Release, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// CheckForUpdateAsync performs an update check in the background and returns immediately.
// If an update is available, it returns the update info through the channel.
func CheckForUpdateAsync(ctx context.Context, dataDir string) <-chan *UpdateInfo {
	ch := make(chan *UpdateInfo, 1)

	go func() {
		defer close(ch)

		// Perform the check.
		info, err := CheckForUpdate(ctx)
		if err != nil {
			// Log error but don't fail.
			fmt.Fprintf(os.Stderr, "Failed to check for updates: %v\n", err)
			return
		}

		// Send update info if available.
		if info.Available {
			ch <- info
		}
	}()

	return ch
}
