package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/version"
	"github.com/google/uuid"
)

type controllerV1 struct {
	*Server
}

func (c *controllerV1) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *controllerV1) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	jsonEncode(w, proto.VersionInfo{
		Version:   version.Version,
		Commit:    version.Commit,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})
}

func (c *controllerV1) handlePostControl(w http.ResponseWriter, r *http.Request) {
	var req proto.ServerControl
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}

	switch req.Command {
	case "shutdown":
		go func() {
			slog.Info("shutting down server...")
			if err := c.Shutdown(context.Background()); err != nil {
				c.logError(r, "failed to shutdown server", "error", err)
			}
		}()
	default:
		c.logError(r, "unknown command", "command", req.Command)
		jsonError(w, http.StatusBadRequest, "unknown command")
		return
	}
}

func (c *controllerV1) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	jsonEncode(w, c.cfg)
}

func (c *controllerV1) handleGetInstances(w http.ResponseWriter, r *http.Request) {
	instances := []proto.Instance{}
	for _, ins := range c.instances.Seq2() {
		// TODO: implement pagination?
		instances = append(instances, proto.Instance{
			ID:      ins.id,
			Path:    ins.path,
			YOLO:    ins.cfg.Permissions != nil && ins.cfg.Permissions.SkipRequests,
			DataDir: ins.cfg.Options.DataDirectory,
			Debug:   ins.cfg.Options.Debug,
			Config:  ins.cfg,
		})
	}
	jsonEncode(w, instances)
}

func (c *controllerV1) handleGetInstanceLSPDiagnostics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	var lsp *lsp.Client
	lspName := r.PathValue("lsp")
	for name, client := range ins.LSPClients.Seq2() {
		if name == lspName {
			lsp = client
			break
		}
	}

	if lsp == nil {
		c.logError(r, "LSP client not found", "id", id, "lsp", lspName)
		jsonError(w, http.StatusNotFound, "LSP client not found")
		return
	}

	diagnostics := lsp.GetDiagnostics()
	jsonEncode(w, diagnostics)
}

func (c *controllerV1) handleGetInstanceLSPs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	lspClients := ins.GetLSPStates()
	jsonEncode(w, lspClients)
}

func (c *controllerV1) handleGetInstanceAgentSessionPromptQueued(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	queued := ins.App.CoderAgent.QueuedPrompts(sid)
	jsonEncode(w, queued)
}

func (c *controllerV1) handlePostInstanceAgentSessionPromptClear(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	ins.App.CoderAgent.ClearQueue(sid)
}

func (c *controllerV1) handleGetInstanceAgentSessionSummarize(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	if err := ins.App.CoderAgent.Summarize(r.Context(), sid); err != nil {
		c.logError(r, "failed to summarize session", "error", err, "id", id, "sid", sid)
		jsonError(w, http.StatusInternalServerError, "failed to summarize session")
		return
	}
}

func (c *controllerV1) handlePostInstanceAgentSessionCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	if ins.App.CoderAgent != nil {
		ins.App.CoderAgent.Cancel(sid)
	}
}

func (c *controllerV1) handleGetInstanceAgentSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	se, err := ins.App.Sessions.Get(r.Context(), sid)
	if err != nil {
		c.logError(r, "failed to get session", "error", err, "id", id, "sid", sid)
		jsonError(w, http.StatusInternalServerError, "failed to get session")
		return
	}

	var isSessionBusy bool
	if ins.App.CoderAgent != nil {
		isSessionBusy = ins.App.CoderAgent.IsSessionBusy(sid)
	}

	jsonEncode(w, proto.AgentSession{
		Session: se,
		IsBusy:  isSessionBusy,
	})
}

func (c *controllerV1) handlePostInstanceAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	w.Header().Set("Accept", "application/json")

	var msg proto.AgentMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		c.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}

	if ins.App.CoderAgent == nil {
		c.logError(r, "coder agent not initialized", "id", id)
		jsonError(w, http.StatusBadRequest, "coder agent not initialized")
		return
	}

	// NOTE: This needs to be on the server's context because the agent runs
	// the request asynchronously.
	// TODO: Look into this one more and make it work synchronously.
	if _, err := ins.App.CoderAgent.Run(c.ctx, msg.SessionID, msg.Prompt, msg.Attachments...); err != nil {
		c.logError(r, "failed to enqueue message", "error", err, "id", id, "sid", msg.SessionID)
		jsonError(w, http.StatusInternalServerError, "failed to enqueue message")
		return
	}
}

func (c *controllerV1) handleGetInstanceAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	var agentInfo proto.AgentInfo
	if ins.App.CoderAgent != nil {
		agentInfo = proto.AgentInfo{
			Model:  ins.App.CoderAgent.Model(),
			IsBusy: ins.App.CoderAgent.IsBusy(),
		}
	}
	jsonEncode(w, agentInfo)
}

func (c *controllerV1) handlePostInstanceAgentUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	if err := ins.App.UpdateAgentModel(); err != nil {
		c.logError(r, "failed to update agent model", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to update agent model")
		return
	}
}

func (c *controllerV1) handlePostInstanceAgentInit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	if err := ins.App.InitCoderAgent(); err != nil {
		c.logError(r, "failed to initialize coder agent", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to initialize coder agent")
		return
	}
}

func (c *controllerV1) handleGetInstanceSessionHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	historyItems, err := ins.App.History.ListBySession(r.Context(), sid)
	if err != nil {
		c.logError(r, "failed to list history", "error", err, "id", id, "sid", sid)
		jsonError(w, http.StatusInternalServerError, "failed to list history")
		return
	}

	jsonEncode(w, historyItems)
}

func (c *controllerV1) handleGetInstanceSessionMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	messages, err := ins.App.Messages.List(r.Context(), sid)
	if err != nil {
		c.logError(r, "failed to list messages", "error", err, "id", id, "sid", sid)
		jsonError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}

	jsonEncode(w, messages)
}

func (c *controllerV1) handleGetInstanceSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sid := r.PathValue("sid")
	session, err := ins.App.Sessions.Get(r.Context(), sid)
	if err != nil {
		c.logError(r, "failedto get session", "error", err, "id", id, "sid", sid)
		jsonError(w, http.StatusInternalServerError, "failed to get session")
		return
	}

	jsonEncode(w, session)
}

func (c *controllerV1) handlePostInstanceSessions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	var args session.Session
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		c.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}

	sess, err := ins.App.Sessions.Create(r.Context(), args.Title)
	if err != nil {
		c.logError(r, "failed to create session", "error", err, "id", id)
		jsonError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	jsonEncode(w, sess)
}

func (c *controllerV1) handleGetInstanceSessions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	sessions, err := ins.App.Sessions.List(r.Context())
	if err != nil {
		c.logError(r, "failed to list sessions", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	jsonEncode(w, sessions)
}

func (c *controllerV1) handlePostInstancePermissionsGrant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	var req proto.PermissionGrant
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}

	switch req.Action {
	case proto.PermissionAllow:
		ins.App.Permissions.Grant(req.Permission)
	case proto.PermissionAllowForSession:
		ins.App.Permissions.GrantPersistent(req.Permission)
	case proto.PermissionDeny:
		ins.App.Permissions.Deny(req.Permission)
	default:
		c.logError(r, "invalid permission action", "action", req.Action)
		jsonError(w, http.StatusBadRequest, "invalid permission action")
		return
	}
}

func (c *controllerV1) handlePostInstancePermissionsSkip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	var req proto.PermissionSkipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}

	ins.App.Permissions.SetSkipRequests(req.Skip)
}

func (c *controllerV1) handleGetInstancePermissionsSkip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	skip := ins.App.Permissions.SkipRequests()
	jsonEncode(w, proto.PermissionSkipRequest{Skip: skip})
}

func (c *controllerV1) handleGetInstanceProviders(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	providers, _ := config.Providers(ins.cfg)
	jsonEncode(w, providers)
}

func (c *controllerV1) handleGetInstanceEvents(w http.ResponseWriter, r *http.Request) {
	flusher := http.NewResponseController(w)
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-r.Context().Done():
			c.logDebug(r, "stopping event stream")
			return
		case ev := <-ins.App.Events():
			c.logDebug(r, "sending event", "event", fmt.Sprintf("%T %+v", ev, ev))
			data, err := json.Marshal(ev)
			if err != nil {
				c.logError(r, "failed to marshal event", "error", err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (c *controllerV1) handleGetInstanceConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	jsonEncode(w, ins.cfg)
}

func (c *controllerV1) handleDeleteInstances(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if ok {
		ins.App.Shutdown()
	}
	c.instances.Del(id)
}

func (c *controllerV1) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ins, ok := c.instances.Get(id)
	if !ok {
		c.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	jsonEncode(w, proto.Instance{
		ID:      ins.id,
		Path:    ins.path,
		YOLO:    ins.cfg.Permissions != nil && ins.cfg.Permissions.SkipRequests,
		DataDir: ins.cfg.Options.DataDirectory,
		Debug:   ins.cfg.Options.Debug,
		Config:  ins.cfg,
	})
}

func (c *controllerV1) handlePostInstances(w http.ResponseWriter, r *http.Request) {
	var args proto.Instance
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		c.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}

	if args.Path == "" {
		c.logError(r, "path is required")
		jsonError(w, http.StatusBadRequest, "path is required")
		return
	}

	id := uuid.New().String()
	cfg, err := config.Init(args.Path, args.DataDir, args.Debug, args.Env)
	if err != nil {
		c.logError(r, "failed to initialize config", "error", err)
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("failed to initialize config: %v", err))
		return
	}

	if cfg.Permissions == nil {
		cfg.Permissions = &config.Permissions{}
	}
	cfg.Permissions.SkipRequests = args.YOLO

	if err := createDotCrushDir(cfg.Options.DataDirectory); err != nil {
		c.logError(r, "failed to create data directory", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to create data directory")
		return
	}

	// Connect to DB; this will also run migrations.
	conn, err := db.Connect(c.ctx, cfg.Options.DataDirectory)
	if err != nil {
		c.logError(r, "failed to connect to database", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to connect to database")
		return
	}

	appInstance, err := app.New(c.ctx, conn, cfg)
	if err != nil {
		slog.Error("failed to create app instance", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to create app instance")
		return
	}

	ins := &Instance{
		App:  appInstance,
		id:   id,
		path: args.Path,
		cfg:  cfg,
		env:  args.Env,
	}

	c.instances.Set(id, ins)
	jsonEncode(w, proto.Instance{
		ID:      id,
		Path:    args.Path,
		DataDir: cfg.Options.DataDirectory,
		Debug:   cfg.Options.Debug,
		YOLO:    cfg.Permissions.SkipRequests,
		Config:  cfg,
		Env:     args.Env,
	})
}

func createDotCrushDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create data directory: %q %w", dir, err)
	}

	gitIgnorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitIgnorePath, []byte("*\n"), 0o644); err != nil {
			return fmt.Errorf("failed to create .gitignore file: %q %w", gitIgnorePath, err)
		}
	}

	return nil
}

func jsonEncode(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(proto.Error{Message: message})
}
