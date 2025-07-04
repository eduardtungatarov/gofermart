package server

import (
	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type Server struct {
	cfg config.Config
	h *handlers.Handler
	m *middleware.Middleware
}

func NewServer(cfg config.Config, h *handlers.Handler, m *middleware.Middleware) *Server {
	return &Server{
		cfg: cfg,
		h: h,
		m: m,
	}
}

func (s *Server) Run() error {
	r := s.getRouter()
	return http.ListenAndServe(s.cfg.RunADDR, r)
}

func (s *Server) getRouter() chi.Router {
	r := chi.NewRouter()

	withJSONReqCheck := r.Group(func(r chi.Router) {
		r.Use(s.m.WithJSONReqCheck)
	})

	withJSONReqCheck.Group(func(r chi.Router) {
		r.Post(
			"/api/user/register",
			s.h.PostUserRegister,
		)
	})

	return r
}
