package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/eduardtungatarov/gofermart/internal/service/auth"

	userRepository "github.com/eduardtungatarov/gofermart/internal/repository/user"

	"go.uber.org/zap"
)

type AuthService interface {
	Register(ctx context.Context, login, pwd string) (string, error)
	Login(ctx context.Context, login, pwd string) (string, error)
}

type Handler struct {
	log         *zap.SugaredLogger
	authService AuthService
}

func MakeHandler(log *zap.SugaredLogger, authService AuthService) *Handler {
	return &Handler{
		log:         log,
		authService: authService,
	}
}

func (h *Handler) PostUserRegister(res http.ResponseWriter, req *http.Request) {
	var reqStr struct {
		Login string `json:"login"`
		Pwd   string `json:"password"`
	}

	defer req.Body.Close()
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqStr); err != nil {
		h.log.Infof("decode request body: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if reqStr.Login == "" || reqStr.Pwd == "" {
		h.log.Infof("login or pwd not be empty")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.authService.Register(req.Context(), reqStr.Login, reqStr.Pwd)
	if err != nil {
		if errors.Is(err, userRepository.ErrUserAlreadyExists) {
			res.WriteHeader(http.StatusConflict)
			return
		}
		h.log.Infof("h.authService.Register: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Authorization", fmt.Sprintf("ApiKey %s", token))
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) PostUserLogin(res http.ResponseWriter, req *http.Request) {
	var reqStr struct {
		Login string `json:"login"`
		Pwd   string `json:"password"`
	}

	defer req.Body.Close()
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqStr); err != nil {
		h.log.Infof("decode request body: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if reqStr.Login == "" || reqStr.Pwd == "" {
		h.log.Infof("login or pwd not be empty")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(req.Context(), reqStr.Login, reqStr.Pwd)
	if err != nil {
		if errors.Is(err, auth.ErrLoginPwd) {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.log.Infof("h.authService.Login: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Authorization", fmt.Sprintf("ApiKey %s", token))
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) PostUserOrders(res http.ResponseWriter, req *http.Request) {

}
