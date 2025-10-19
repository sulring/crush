package hyper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	fantasy "charm.land/fantasy"
)

// Client is a minimal client for the Hyper API.
type Client struct {
	BaseURL    *url.URL
	APIKey     string
	HTTPClient *http.Client
}

// New creates a new Hyper client.
func New(base string, apiKey string) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	return &Client{
		BaseURL:    u,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Project mirrors the JSON returned by Hyper.
type Project struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	UserID         uuid.UUID  `json:"user_id"`
	Archived       bool       `json:"archived"`
	Identifiers    []string   `json:"identifiers"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// CreateProject creates a project.
func (c *Client) CreateProject(ctx context.Context, name, description string, organizationID uuid.UUID, identifiers []string) (Project, error) {
	var p Project
	body := map[string]any{
		"name":            name,
		"description":     description,
		"organization_id": organizationID,
	}
	if len(identifiers) > 0 {
		body["identifiers"] = identifiers
	}
	bts, _ := json.Marshal(body)
	endpoint := c.BaseURL.JoinPath("api/v1", "projects").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bts))
	if err != nil {
		return p, err
	}
	c.addAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return p, fmt.Errorf("create project: http %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return p, err
	}
	return p, nil
}

// ListProjects lists projects for the authenticated user.
// If identifiers is not empty, projects that match ANY identifier are returned.
func (c *Client) ListProjects(ctx context.Context, identifiers []string) ([]Project, error) {
	endpoint := c.BaseURL.JoinPath("api/v1", "projects")
	q := endpoint.Query()
	if len(identifiers) > 0 {
		q.Set("identifiers", strings.Join(identifiers, ","))
		endpoint.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list projects: http %d", resp.StatusCode)
	}
	var ps []Project
	if err := json.NewDecoder(resp.Body).Decode(&ps); err != nil {
		return nil, err
	}
	return ps, nil
}

// Memorize sends messages to be memorized for a given project and echoes them back.
func (c *Client) Memorize(ctx context.Context, projectID uuid.UUID, msgs []fantasy.Message) ([]fantasy.Message, error) {
	bts, _ := json.Marshal(msgs)
	endpoint := c.BaseURL.JoinPath("api/v1", "projects", projectID.String(), "memorize").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bts))
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("memorize: http %d", resp.StatusCode)
	}
	var echoed []fantasy.Message
	if err := json.NewDecoder(resp.Body).Decode(&echoed); err != nil {
		return nil, err
	}
	return echoed, nil
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) addAuth(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
}
