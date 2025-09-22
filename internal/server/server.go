package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
)

// ErrServerClosed is returned when the server is closed.
var ErrServerClosed = fmt.Errorf("server closed")

// InstanceState represents the state of a running [app.App] instance.
type InstanceState uint8

const (
	// InstanceStateCreated indicates that the instance has been created but not yet started.
	InstanceStateCreated InstanceState = iota
	// InstanceStateStarted indicates that the instance is currently running.
	InstanceStateStarted
	// InstanceStateStopped indicates that the instance has been stopped.
	InstanceStateStopped
)

// Instance represents a running [app.App] instance with its associated
// resources and state.
type Instance struct {
	*app.App
	State InstanceState
	ln    net.Listener
	cfg   *config.Config
	id    string
	path  string
}

// ID returns the unique identifier of the instance.
func (i *Instance) ID() string {
	return i.id
}

// Path returns the filesystem path associated with the instance.
func (i *Instance) Path() string {
	return i.path
}

// DefaultAddr returns the default address path for the Crush server based on
// the operating system.
func DefaultAddr() string {
	sockPath := "crush.sock"
	user, err := user.Current()
	if err == nil && user.Uid != "" {
		sockPath = fmt.Sprintf("crush-%s.sock", user.Uid)
	}
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\%s`, sockPath)
	}
	return filepath.Join(os.TempDir(), sockPath)
}

// Server represents a Crush server instance bound to a specific address.
type Server struct {
	// Addr can be a TCP address, a Unix socket path, or a Windows named pipe.
	Addr string

	h  *http.Server
	ln net.Listener

	// instances is a map of running applications managed by the server.
	instances *csync.Map[string, *Instance]
	cfg       *config.Config
	logger    *slog.Logger
}

// SetLogger sets the logger for the server.
func (s *Server) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

// DefaultServer returns a new [Server] instance with the default address.
func DefaultServer(cfg *config.Config) *Server {
	return NewServer(cfg, "unix", DefaultAddr())
}

// NewServer is a helper to create a new [Server] instance with the given
// address. On Windows, if the address is not a "tcp" address, it will be
// converted to a named pipe format.
func NewServer(cfg *config.Config, network, address string) *Server {
	if runtime.GOOS == "windows" && !strings.HasPrefix(address, "tcp") &&
		!strings.HasPrefix(address, `\\.\pipe\`) {
		// On Windows, convert to named pipe format if not TCP
		// (e.g., "mypipe" -> "\\.\pipe\mypipe")
		address = fmt.Sprintf(`\\.\pipe\%s`, address)
	}

	s := new(Server)
	s.Addr = address
	s.cfg = cfg
	s.instances = csync.NewMap[string, *Instance]()

	var p http.Protocols
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", s.handleGetConfig)
	mux.HandleFunc("GET /v1/instances", s.handleGetInstances)
	mux.HandleFunc("POST /v1/instances", s.handlePostInstances)
	mux.HandleFunc("DELETE /v1/instances", s.handleDeleteInstances)
	mux.HandleFunc("GET /v1/instances/{id}/events", s.handleGetInstanceEvents)
	s.h = &http.Server{
		Protocols: &p,
		Handler:   s.loggingHandler(mux),
	}
	return s
}

// Serve accepts incoming connections on the listener.
func (s *Server) Serve(ln net.Listener) error {
	return s.h.Serve(ln)
}

// ListenAndServe starts the server and begins accepting connections.
func (s *Server) ListenAndServe() error {
	if s.ln != nil {
		return fmt.Errorf("server already started")
	}
	ln, err := listen("unix", s.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.Addr, err)
	}
	return s.Serve(ln)
}

func (s *Server) closeListener() {
	if s.ln != nil {
		s.ln.Close()
		s.ln = nil
	}
}

// Close force close all listeners and connections.
func (s *Server) Close() error {
	defer func() { s.closeListener() }()
	return s.h.Close()
}

// Shutdown gracefully shuts down the server without interrupting active
// connections. It stops accepting new connections and waits for existing
// connections to finish.
func (s *Server) Shutdown(ctx context.Context) error {
	defer func() { s.closeListener() }()
	return s.h.Shutdown(ctx)
}

func (s *Server) logError(r *http.Request, msg string, args ...any) {
	if s.logger != nil {
		s.logger.With(
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("remote_addr", r.RemoteAddr),
		).Error(msg, args...)
	}
}
