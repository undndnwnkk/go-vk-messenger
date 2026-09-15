package handler

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) http.Handler {
	h := &Handler{pool: pool}
	r := chi.NewRouter()

	r.Get("/health", h.healthHandler)

	return r
}

func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string, 1)
	if err := h.pool.Ping(r.Context()); err != nil {
		resp["status"] = "unvailable"
		writeJSON(w, 500, resp)
	} else {
		resp["status"] = "ok"
		writeJSON(w, 200, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(&data)
}
