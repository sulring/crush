package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/charmbracelet/crush/internal/config"
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
	}
	hasher := sha256.New()
	hasher.Write([]byte(path))
	id := hex.EncodeToString(hasher.Sum(nil))
	return &Client{
		h:    h,
		id:   id,
		path: filepath.Clean(path),
	}, nil
}

// ID returns the client's instance unique identifier.
func (c *Client) ID() string {
	return c.id
}

// Path returns the client's instance filesystem path.
func (c *Client) Path() string {
	return c.path
}

// GetConfig retrieves the server's configuration via RPC.
func (c *Client) GetConfig() (*config.Config, error) {
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
