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

// DummyHost is used to satisfy the http.Client's requirement for a URL.
const DummyHost = "api.crush.localhost"

// Client represents an RPC client connected to a Crush server.
type Client struct {
	h     *http.Client
	id    string
	path  string
	proto string
	addr  string
}

// DefaultClient creates a new [Client] connected to the default server address.
func DefaultClient(path string) (*Client, error) {
	host, err := server.ParseHostURL(server.DefaultHost())
	if err != nil {
		return nil, err
	}
	return NewClient(path, host.Scheme, host.Host)
}

// NewClient creates a new [Client] connected to the server at the given
// network and address.
func NewClient(path, network, address string) (*Client, error) {
	c := new(Client)
	c.path = filepath.Clean(path)
	c.proto = network
	c.addr = address
	p := &http.Protocols{}
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Protocols = p
	tr.DialContext = c.dialer
	if c.proto == "npipe" || c.proto == "unix" {
		// We don't need compression for local connections.
		tr.DisableCompression = true
	}
	c.h = &http.Client{
		Transport: tr,
		Timeout:   0, // we need this to be 0 for long-lived connections and SSE streams
	}
	return c, nil
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

// ShutdownServer sends a shutdown request to the server.
func (c *Client) ShutdownServer() error {
	req, err := http.NewRequest("POST", "http://localhost/v1/control", jsonBody(proto.ServerControl{
		Command: "shutdown",
	}))
	if err != nil {
		return err
	}
	rsp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("server shutdown failed: %s", rsp.Status)
	}
	return nil
}

func (c *Client) dialer(ctx context.Context, network, address string) (net.Conn, error) {
	d := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	switch c.proto {
	case "npipe":
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return dialPipeContext(ctx, c.addr)
	case "unix":
		return d.DialContext(ctx, "unix", c.addr)
	default:
		return d.DialContext(ctx, network, address)
	}
}
