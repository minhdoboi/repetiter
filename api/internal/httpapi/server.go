package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5"

	"repetiter/internal/config"
	"repetiter/internal/providers"
	"repetiter/internal/store"
)

type Server struct {
	cfg   config.Config
	store *store.Store
	prov  *providers.Registry
	http  http.Handler
}

func New(cfg config.Config, st *store.Store, prov *providers.Registry) *Server {
	s := &Server{cfg: cfg, store: st, prov: prov}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.AllowAll().Handler)

	r.Get("/health", s.health)
	r.Route("/v1", func(r chi.Router) {
		r.Use(s.auth)
		r.Get("/me", s.getMe)
		r.Patch("/me", s.patchMe)
		r.Post("/sentences", s.createSentence)
		r.Get("/sentences", s.listSentences)
		r.Delete("/sentences/history", s.clearHistory)
		r.Get("/sentences/{id}", s.getSentence)
		r.Patch("/sentences/{id}", s.patchSentence)
		r.Delete("/sentences/{id}", s.deleteSentence)
		r.Get("/folders", s.listFolders)
		r.Post("/folders", s.createFolder)
		r.Post("/tts", s.tts)
		r.Get("/tts/voices", s.listTTSVoices)
	})
	s.http = r
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.http.ServeHTTP(w, r)
}

type ctxKey string

const userIDKey ctxKey = "userID"

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := strings.TrimSpace(s.cfg.DevUserID)
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "sign in required (set DEV_USER_ID locally, or Clerk later)")
			return
		}
		u, err := s.store.UpsertUser(r.Context(), userID, s.cfg.DefaultTextProv, s.cfg.DefaultTextModel, s.cfg.DefaultTTSProv, s.cfg.DefaultTTSModel, s.cfg.DefaultTTSVoice)
		if err != nil {
			log.Printf("upsert user: %v", err)
			writeError(w, http.StatusInternalServerError, "could not load account")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, u.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userID(r *http.Request) string {
	id, _ := r.Context().Value(userIDKey).(string)
	return id
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

var errNotFound = errors.New("not found")

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errNotFound) || errors.Is(err, store.ErrNotFound)
}
