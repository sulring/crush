package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/user"
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
	return fmt.Sprintf("/tmp/%s", sockPath)
}

// Server represents a Crush server instance bound to a specific address.
type Server struct {
	// Addr can be a TCP address, a Unix socket path, or a Windows named pipe.
	Addr string

	h   *http.Server
	ln  net.Listener
	ctx context.Context

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
	s.ctx = context.Background()

	var p http.Protocols
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	c := &controllerV1{Server: s}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", c.handleGetHealth)
	mux.HandleFunc("GET /v1/version", c.handleGetVersion)
	mux.HandleFunc("GET /v1/config", c.handleGetConfig)
	mux.HandleFunc("GET /v1/instances", c.handleGetInstances)
	mux.HandleFunc("POST /v1/instances", c.handlePostInstances)
	mux.HandleFunc("DELETE /v1/instances", c.handleDeleteInstances)
	mux.HandleFunc("GET /v1/instances/{id}/config", c.handleGetInstanceConfig)
	mux.HandleFunc("GET /v1/instances/{id}/events", c.handleGetInstanceEvents)
	mux.HandleFunc("GET /v1/instances/{id}/sessions", c.handleGetInstanceSessions)
	mux.HandleFunc("POST /v1/instances/{id}/sessions", c.handlePostInstanceSessions)
	mux.HandleFunc("GET /v1/instances/{id}/sessions/{sid}", c.handleGetInstanceSession)
	mux.HandleFunc("GET /v1/instances/{id}/sessions/{sid}/history", c.handleGetInstanceSessionHistory)
	mux.HandleFunc("GET /v1/instances/{id}/sessions/{sid}/messages", c.handleGetInstanceSessionMessages)
	mux.HandleFunc("GET /v1/instances/{id}/lsps", c.handleGetInstanceLSPs)
	mux.HandleFunc("GET /v1/instances/{id}/lsps/{lsp}/diagnostics", c.handleGetInstanceLSPDiagnostics)
	mux.HandleFunc("GET /v1/instances/{id}/permissions/skip", c.handleGetInstancePermissionsSkip)
	mux.HandleFunc("POST /v1/instances/{id}/permissions/skip", c.handlePostInstancePermissionsSkip)
	mux.HandleFunc("POST /v1/instances/{id}/permissions/grant", c.handlePostInstancePermissionsGrant)
	mux.HandleFunc("GET /v1/instances/{id}/agent", c.handleGetInstanceAgent)
	mux.HandleFunc("POST /v1/instances/{id}/agent", c.handlePostInstanceAgent)
	mux.HandleFunc("POST /v1/instances/{id}/agent/init", c.handlePostInstanceAgentInit)
	mux.HandleFunc("POST /v1/instances/{id}/agent/update", c.handlePostInstanceAgentUpdate)
	mux.HandleFunc("GET /v1/instances/{id}/agent/sessions/{sid}", c.handleGetInstanceAgentSession)
	mux.HandleFunc("POST /v1/instances/{id}/agent/sessions/{sid}/cancel", c.handlePostInstanceAgentSessionCancel)
	mux.HandleFunc("GET /v1/instances/{id}/agent/sessions/{sid}/prompts/queued", c.handleGetInstanceAgentSessionPromptQueued)
	mux.HandleFunc("POST /v1/instances/{id}/agent/sessions/{sid}/prompts/clear", c.handlePostInstanceAgentSessionPromptClear)
	mux.HandleFunc("POST /v1/instances/{id}/agent/sessions/{sid}/summarize", c.handleGetInstanceAgentSessionSummarize)
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

func (s *Server) logDebug(r *http.Request, msg string, args ...any) {
	if s.logger != nil {
		s.logger.With(
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("remote_addr", r.RemoteAddr),
		).Debug(msg, args...)
	}
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
