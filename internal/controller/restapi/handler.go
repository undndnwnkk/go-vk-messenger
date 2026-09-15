package handler

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

type Handler struct{}

func NewHandler() *http.Handler {
	h := &Handler{}
	r := chi.NewRouter()
}
