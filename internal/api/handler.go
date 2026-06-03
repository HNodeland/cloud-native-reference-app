package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/HNodeland/cloud-native-reference-app/internal/store"
	"github.com/HNodeland/cloud-native-reference-app/internal/telemetry"
	"go.uber.org/zap"
)

type handler struct {
	store *store.Store
}

func RegisterRoutes(mux *http.ServeMux, s *store.Store, logger *zap.Logger, telemetryMiddleware *telemetry.Instrumentation) {
	h := &handler{store: s}
	todosHandler := http.HandlerFunc(h.route)
	todosHandler = loggingMiddleware(logger, telemetryMiddleware.HTTPMiddleware(todosHandler))

	mux.Handle("/todos", todosHandler)
	mux.Handle("/todos/", todosHandler)
}

func (h *handler) route(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/todos":
		h.handleTodos(w, r)
	case strings.HasPrefix(path, "/todos/"):
		h.handleTodoByID(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *handler) handleTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.List())
	case http.MethodPost:
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Title) == "" {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, h.store.Create(strings.TrimSpace(req.Title)))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *handler) handleTodoByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/todos/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		todo, ok := h.store.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, todo)
	case http.MethodPut:
		var req store.UpdateInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
			http.Error(w, "title must not be empty", http.StatusBadRequest)
			return
		}
		if req.Title != nil {
			trimmed := strings.TrimSpace(*req.Title)
			req.Title = &trimmed
		}
		todo, ok := h.store.Update(id, req)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, todo)
	case http.MethodDelete:
		if !h.store.Delete(id) {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func loggingMiddleware(logger *zap.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		logger.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rw.statusCode),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
