package server

import (
	"context"
	"net/http"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	cfg config.Config
	h   *handlers.Handler
	m   *middleware.Middleware
}

func NewServer(cfg config.Config, h *handlers.Handler, m *middleware.Middleware) *Server {
	return &Server{
		cfg: cfg,
		h:   h,
		m:   m,
	}
}

func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.cfg.RunADDR,
		Handler: s.GetRouter(),
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		// Даем возможность серверу в течение 5 сек завершиться корректно.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTime)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func (s *Server) GetRouter() chi.Router {
	r := chi.NewRouter()

	r.With(s.m.WithJSONReqCheck).Post(
		"/api/user/register",
		s.h.PostUserRegister,
	)
	r.With(s.m.WithJSONReqCheck).Post(
		"/api/user/login",
		s.h.PostUserLogin,
	)

	r.Group(func(r chi.Router) {
		r.Use(s.m.WithAuth)
		r.With(s.m.WithTextPlainReqCheck).Post(
			"/api/user/orders",
			s.h.PostUserOrders,
		)
		r.Get(
			"/api/user/orders",
			s.h.GetUserOrders,
		)
		r.Get(
			"/api/user/balance",
			s.h.GetUserBalance,
		)
		r.Get(
			"/api/user/balance/withdraw",
			s.h.GetUserBalanceWithdraw,
		)
		r.With(s.m.WithJSONReqCheck).Post(
			"/api/user/balance/withdraw",
			s.h.PostUserBalanceWithdraw,
		)
	})

	return r
}
