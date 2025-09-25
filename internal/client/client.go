package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	stdpath "path"
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
	h       *http.Client
	path    string
	network string
	addr    string
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
	c.network = network
	c.addr = address
	p := &http.Protocols{}
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Protocols = p
	tr.DialContext = c.dialer
	if c.network == "npipe" || c.network == "unix" {
		// We don't need compression for local connections.
		tr.DisableCompression = true
	}
	c.h = &http.Client{
		Transport: tr,
		Timeout:   0, // we need this to be 0 for long-lived connections and SSE streams
	}
	return c, nil
}

// Path returns the client's instance filesystem path.
func (c *Client) Path() string {
	return c.path
}

// GetGlobalConfig retrieves the server's configuration.
func (c *Client) GetGlobalConfig(ctx context.Context) (*config.Config, error) {
	var cfg config.Config
	rsp, err := c.get(ctx, "/config", nil, nil)
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
func (c *Client) Health(ctx context.Context) error {
	rsp, err := c.get(ctx, "/health", nil, nil)
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
func (c *Client) VersionInfo(ctx context.Context) (*proto.VersionInfo, error) {
	var vi proto.VersionInfo
	rsp, err := c.get(ctx, "version", nil, nil)
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
func (c *Client) ShutdownServer(ctx context.Context) error {
	rsp, err := c.post(ctx, "/control", nil, jsonBody(proto.ServerControl{
		Command: "shutdown",
	}), nil)
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
	// It's important to use the client's addr for npipe/unix and not the
	// address param because the address param is always "localhost:port" for
	// HTTP clients and npipe/unix don't have a concept of ports.
	switch c.network {
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

func (c *Client) get(ctx context.Context, path string, query url.Values, headers http.Header) (*http.Response, error) {
	return c.sendReq(ctx, http.MethodGet, path, query, nil, headers)
}

func (c *Client) post(ctx context.Context, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error) {
	return c.sendReq(ctx, http.MethodPost, path, query, body, headers)
}

func (c *Client) put(ctx context.Context, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error) {
	return c.sendReq(ctx, http.MethodPut, path, query, body, headers)
}

func (c *Client) delete(ctx context.Context, path string, query url.Values, headers http.Header) (*http.Response, error) {
	return c.sendReq(ctx, http.MethodDelete, path, query, nil, headers)
}

func (c *Client) sendReq(ctx context.Context, method, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error) {
	url := (&url.URL{
		Path:     stdpath.Join("/v1", path), // Right now, we only have v1
		RawQuery: query.Encode(),
	}).String()
	req, err := c.buildReq(ctx, method, url, body, headers)
	if err != nil {
		return nil, err
	}

	rsp, err := c.doReq(req)
	if err != nil {
		return nil, err
	}

	// TODO: check server errors in the response body?

	return rsp, nil
}

func (c *Client) doReq(req *http.Request) (*http.Response, error) {
	rsp, err := c.h.Do(req)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

func (c *Client) buildReq(ctx context.Context, method, url string, body io.Reader, headers http.Header) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		r.Header[http.CanonicalHeaderKey(k)] = v
	}

	r.URL.Scheme = "http" // This is always http because we don't use TLS
	r.URL.Host = c.addr
	if c.network == "npipe" || c.network == "unix" {
		// We use a dummy host for non-tcp connections.
		r.Host = DummyHost
	}

	if body != nil && r.Header.Get("Content-Type") == "" {
		r.Header.Set("Content-Type", "text/plain")
	}

	return r, nil
}
