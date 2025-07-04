package handlers

import (
	"go.uber.org/zap"
	"net/http"
)

type Handler struct {
	log *zap.SugaredLogger
}

func MakeHandler(log *zap.SugaredLogger) *Handler {
	return &Handler{
		log: log,
	}
}

func (h *Handler) PostUserRegister(res http.ResponseWriter, req *http.Request) {

}
