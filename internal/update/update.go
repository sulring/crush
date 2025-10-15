package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/crush/internal/version"
)

const (
	githubAPIURL = "https://api.github.com/repos/charmbracelet/crush/releases/latest"
	userAgent    = "crush/1.0"
)

// Info contains information about an available update.
type Info struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	Available      bool
}

// Check checks if a new version is available.
func Check(ctx context.Context) (*Info, error) {
	info := &Info{
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

// githubRelease represents a GitHub release.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// fetchLatestRelease fetches the latest release information from GitHub.
func fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
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

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}
