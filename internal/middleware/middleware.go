package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/eduardtungatarov/gofermart/internal/config"

	"go.uber.org/zap"
)

//go:generate mockery --name=AuthService
type AuthService interface {
	GetUserIDByToken(tokenStr string) (int, error)
}

type Middleware struct {
	log         *zap.SugaredLogger
	authService AuthService
}

func MakeMiddleware(log *zap.SugaredLogger, authService AuthService) *Middleware {
	return &Middleware{
		log:         log,
		authService: authService,
	}
}

func (m *Middleware) WithJSONReqCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
			m.log.Info("Ожидался json тип запроса")
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		next.ServeHTTP(res, req)
	})
}

func (m *Middleware) WithTextPlainReqCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !strings.Contains(req.Header.Get("Content-Type"), "text/plain") {
			m.log.Info("Ожидался text/plain тип запроса")
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		next.ServeHTTP(res, req)
	})
}

func (m *Middleware) WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		token := strings.Replace(authHeader, "Bearer ", "", 1)
		if token == "" {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID, err := m.authService.GetUserIDByToken(token)
		if err != nil {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := req.Context()
		newCtx := context.WithValue(ctx, config.UserIDKeyName, userID)
		req = req.WithContext(newCtx)

		next.ServeHTTP(res, req)
	})
}
