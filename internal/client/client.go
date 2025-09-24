package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/server"
)

// Client represents an RPC client connected to a Crush server.
type Client struct {
	h    *http.Client
	id   string
	path string
}

// DefaultClient creates a new [Client] connected to the default server address.
func DefaultClient(path string) (*Client, error) {
	return NewClient(path, "unix", server.DefaultAddr())
}

// NewClient creates a new [Client] connected to the server at the given
// network and address.
func NewClient(path, network, address string) (*Client, error) {
	var p http.Protocols
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Protocols = &p
	tr.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		d := net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		return d.DialContext(ctx, network, address)
	}
	h := &http.Client{
		Transport: tr,
		Timeout:   0, // we need this to be 0 for long-lived connections and SSE streams
	}
	return &Client{
		h:    h,
		path: filepath.Clean(path),
	}, nil
}

// ID returns the client's instance unique identifier.
func (c *Client) ID() string {
	return c.id
}

// SetID sets the client's instance unique identifier.
func (c *Client) SetID(id string) {
	c.id = id
}

// Path returns the client's instance filesystem path.
func (c *Client) Path() string {
	return c.path
}

// GetGlobalConfig retrieves the server's configuration.
func (c *Client) GetGlobalConfig() (*config.Config, error) {
	var cfg config.Config
	rsp, err := c.h.Get("http://localhost/v1/config")
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if err := json.NewDecoder(rsp.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Health checks the server's health status.
func (c *Client) Health() error {
	rsp, err := c.h.Get("http://localhost/v1/health")
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("server health check failed: %s", rsp.Status)
	}
	return nil
}

// VersionInfo retrieves the server's version information.
func (c *Client) VersionInfo() (*proto.VersionInfo, error) {
	var vi proto.VersionInfo
	rsp, err := c.h.Get("http://localhost/v1/version")
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if err := json.NewDecoder(rsp.Body).Decode(&vi); err != nil {
		return nil, err
	}
	return &vi, nil
}
