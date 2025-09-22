package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/proto"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	jsonEncode(w, s.cfg)
}

func (s *Server) handleGetInstances(w http.ResponseWriter, r *http.Request) {
	instances := []proto.Instance{}
	for _, ins := range s.instances.Seq2() {
		instances = append(instances, proto.Instance{
			ID:   ins.ID(),
			Path: ins.Path(),
			YOLO: ins.cfg.Permissions != nil && ins.cfg.Permissions.SkipRequests,
		})
	}
	jsonEncode(w, instances)
}

func (s *Server) handleGetInstanceEvents(w http.ResponseWriter, r *http.Request) {
	flusher := http.NewResponseController(w)
	id := r.PathValue("id")
	ins, ok := s.instances.Get(id)
	if !ok {
		s.logError(r, "instance not found", "id", id)
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ins.App.Events():
			data, err := json.Marshal(ev)
			if err != nil {
				s.logError(r, "failed to marshal event", "error", err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleDeleteInstances(w http.ResponseWriter, r *http.Request) {
	var ids []string
	id := r.URL.Query().Get("id")
	if id != "" {
		ids = append(ids, id)
	}

	// Get IDs from body
	var args []proto.Instance
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		s.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}
	ids = append(ids, func() []string {
		out := make([]string, len(args))
		for i, arg := range args {
			out[i] = arg.ID
		}
		return out
	}()...)

	for _, id := range ids {
		s.instances.Del(id)
	}
}

func (s *Server) handlePostInstances(w http.ResponseWriter, r *http.Request) {
	var args proto.Instance
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		s.logError(r, "failed to decode request", "error", err)
		jsonError(w, http.StatusBadRequest, "failed to decode request")
		return
	}

	ctx := r.Context()
	hasher := sha256.New()
	hasher.Write([]byte(filepath.Clean(args.Path)))
	id := hex.EncodeToString(hasher.Sum(nil))
	if existing, ok := s.instances.Get(id); ok {
		jsonEncode(w, proto.Instance{
			ID:   existing.ID(),
			Path: existing.Path(),
			YOLO: existing.cfg.Permissions != nil && existing.cfg.Permissions.SkipRequests,
		})
		return
	}

	cfg, err := config.Init(args.Path, s.cfg.Options.DataDirectory, s.cfg.Options.Debug)
	if err != nil {
		s.logError(r, "failed to initialize config", "error", err)
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("failed to initialize config: %v", err))
		return
	}

	if cfg.Permissions == nil {
		cfg.Permissions = &config.Permissions{}
	}
	cfg.Permissions.SkipRequests = args.YOLO

	if err := createDotCrushDir(cfg.Options.DataDirectory); err != nil {
		s.logError(r, "failed to create data directory", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to create data directory")
		return
	}

	// Connect to DB; this will also run migrations.
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		s.logError(r, "failed to connect to database", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to connect to database")
		return
	}

	appInstance, err := app.New(ctx, conn, cfg)
	if err != nil {
		slog.Error("failed to create app instance", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to create app instance")
		return
	}

	ins := &Instance{
		App:   appInstance,
		State: InstanceStateCreated,
		id:    id,
		path:  args.Path,
		cfg:   cfg,
	}

	s.instances.Set(id, ins)
	jsonEncode(w, proto.Instance{
		ID:   id,
		Path: args.Path,
		YOLO: cfg.Permissions.SkipRequests,
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
