package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robfig/cron/v3"

	"github.com/durgakar/reward-system-users/internal/app"
	"github.com/durgakar/reward-system-users/internal/domain"
)

type Server struct {
	app    *app.Application
	router chi.Router
	cron   *cron.Cron
	log    *slog.Logger
	runMu  sync.Mutex
}

func New(application *app.Application, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{app: application, log: log, cron: cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))}
	s.routes()
	return s
}

func (s *Server) routes() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(60*time.Second))

	r.Get("/health", s.handleHealth)
	r.Get("/ready", s.handleReady)

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(s.authMiddleware)
		api.Get("/clients", s.handleListClients)
		api.Route("/segments", func(seg chi.Router) {
			seg.Get("/", s.handleListSegments)
			seg.Post("/", s.handleCreateSegment)
			seg.Get("/{id}", s.handleGetSegment)
			seg.Put("/{id}", s.handleUpdateSegment)
			seg.Delete("/{id}", s.handleDeleteSegment)
		})
		api.Route("/rules", func(rule chi.Router) {
			rule.Get("/", s.handleListRules)
			rule.Post("/", s.handleCreateRule)
			rule.Get("/{id}", s.handleGetRule)
			rule.Put("/{id}", s.handleUpdateRule)
			rule.Delete("/{id}", s.handleDeleteRule)
		})
		api.Post("/campaigns/run", s.handleRunCampaign)
		api.Get("/campaigns/runs", s.handleListCampaignRuns)
	})

	r.Get("/admin", s.handleAdmin)
	r.Get("/admin/*", s.handleAdmin)

	s.router = r
}

func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) StartCron(ctx context.Context) {
	if !s.app.Config.Cron.Enabled {
		return
	}
	_, err := s.cron.AddFunc(s.app.Config.Cron.Schedule, func() {
		s.log.Info("cron campaign start", "schedule", s.app.Config.Cron.Schedule)
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		results, err := s.app.RunCampaign(runCtx)
		if err != nil {
			s.log.Error("cron campaign failed", "error", err)
			return
		}
		s.log.Info("cron campaign finished", "clients", len(results))
	})
	if err != nil {
		s.log.Error("invalid cron schedule", "error", err)
		return
	}
	s.cron.Start()
}

func (s *Server) StopCron() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
	}
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	key := s.app.Config.Server.AdminAPIKey
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin API key not configured"})
			return
		}
		auth := r.Header.Get("Authorization")
		if auth == "Bearer "+key || r.Header.Get("X-API-Key") == key {
			next.ServeHTTP(w, r)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	})
}

func (s *Server) requireStore(w http.ResponseWriter) bool {
	if s.app.Store == nil {
		writeErr(w, http.StatusServiceUnavailable, fmt.Errorf("database required"))
		return false
	}
	return true
}

func (s *Server) rebuildAfterWrite(w http.ResponseWriter) bool {
	if err := s.app.RebuildRunner(); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("rebuild runner: %w", err))
		return false
	}
	return true
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.app.Store != nil {
		if err := s.app.Store.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db unavailable"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleListClients(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	clients, err := s.app.Store.ListClients(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

func (s *Server) handleListSegments(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	seg, err := s.app.Store.ListSegments(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, seg)
}

func (s *Server) handleGetSegment(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	id := chi.URLParam(r, "id")
	seg, err := s.app.Store.GetSegment(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, seg)
}

func (s *Server) handleCreateSegment(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	var seg domain.SegmentDefinition
	if err := decodeJSON(r, &seg); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if seg.ID == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id required"))
		return
	}
	if err := s.app.Store.UpsertSegment(r.Context(), seg); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !s.rebuildAfterWrite(w) {
		return
	}
	writeJSON(w, http.StatusCreated, seg)
}

func (s *Server) handleUpdateSegment(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	id := chi.URLParam(r, "id")
	var seg domain.SegmentDefinition
	if err := decodeJSON(r, &seg); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	seg.ID = id
	if err := s.app.Store.UpsertSegment(r.Context(), seg); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !s.rebuildAfterWrite(w) {
		return
	}
	writeJSON(w, http.StatusOK, seg)
}

func (s *Server) handleDeleteSegment(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.app.Store.DeleteSegment(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !s.rebuildAfterWrite(w) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	rules, err := s.app.Store.ListRules(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleGetRule(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	id := chi.URLParam(r, "id")
	rule, err := s.app.Store.GetRule(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	var rule domain.RuleDefinition
	if err := decodeJSON(r, &rule); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if rule.ID == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id required"))
		return
	}
	if err := s.app.Store.UpsertRule(r.Context(), rule); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !s.rebuildAfterWrite(w) {
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	id := chi.URLParam(r, "id")
	var rule domain.RuleDefinition
	if err := decodeJSON(r, &rule); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	rule.ID = id
	if err := s.app.Store.UpsertRule(r.Context(), rule); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !s.rebuildAfterWrite(w) {
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.app.Store.DeleteRule(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !s.rebuildAfterWrite(w) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRunCampaign(w http.ResponseWriter, r *http.Request) {
	if !s.runMu.TryLock() {
		writeErr(w, http.StatusConflict, fmt.Errorf("campaign already running"))
		return
	}
	defer s.runMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	results, err := s.app.RunCampaign(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "count": len(results)})
}

func (s *Server) handleListCampaignRuns(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	runs, err := s.app.Store.ListCampaignRuns(r.Context(), 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
