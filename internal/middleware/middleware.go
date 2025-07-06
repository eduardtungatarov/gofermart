package middleware

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type Middleware struct {
	log *zap.SugaredLogger
}

func MakeMiddleware(log *zap.SugaredLogger) *Middleware {
	return &Middleware{
		log: log,
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
