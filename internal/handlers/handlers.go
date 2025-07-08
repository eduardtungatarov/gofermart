package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"unicode"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal/queries"

	balanceQ "github.com/eduardtungatarov/gofermart/internal/repository/balance/queries"
	orderQ "github.com/eduardtungatarov/gofermart/internal/repository/order/queries"

	"github.com/eduardtungatarov/gofermart/internal/service/order"

	"github.com/eduardtungatarov/gofermart/internal/service/auth"

	userRepository "github.com/eduardtungatarov/gofermart/internal/repository/user"

	"go.uber.org/zap"
)

//go:generate mockery --name=AuthService
type AuthService interface {
	Register(ctx context.Context, login, pwd string) (string, error)
	Login(ctx context.Context, login, pwd string) (string, error)
}

//go:generate mockery --name=OrderService
type OrderService interface {
	PostUserOrders(ctx context.Context, orderNumber string) error
	GetUserOrders(ctx context.Context) ([]orderQ.Order, error)
}

//go:generate mockery --name=BalanceService
type BalanceService interface {
	GetUserBalance(ctx context.Context) (balanceQ.Balance, error)
}

//go:generate mockery --name=WithdrawalService
type WithdrawalService interface {
	GetUserWithdrawals(ctx context.Context) ([]queries.Withdrawal, error)
	SaveWithdrawal(ctx context.Context, orderNumber string, sum int) error
}

type Handler struct {
	log               *zap.SugaredLogger
	authService       AuthService
	orderService      OrderService
	balanceService    BalanceService
	withdrawalService WithdrawalService
}

func MakeHandler(
	log *zap.SugaredLogger,
	authService AuthService,
	orderService OrderService,
	balanceService BalanceService,
	withdrawalService WithdrawalService,
) *Handler {
	return &Handler{
		log:               log,
		authService:       authService,
		orderService:      orderService,
		balanceService:    balanceService,
		withdrawalService: withdrawalService,
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

	res.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
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

	res.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) PostUserOrders(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.log.Infof("Не удалось прочитать тело запроса PostUserOrders: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	if len(body) == 0 {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	orderNumber := string(body)
	if !h.isValidLuhn(orderNumber) {
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	err = h.orderService.PostUserOrders(req.Context(), orderNumber)
	if err != nil {
		if errors.Is(err, order.ErrOrderAlreadyUploadedByUser) {
			res.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, order.ErrOrderAlreadyUploadedByAnotherUser) {
			res.WriteHeader(http.StatusConflict)
			return
		}
		h.log.Infof("h.OrderService.PostUserOrders: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusAccepted)
	return
}

func (h *Handler) GetUserOrders(res http.ResponseWriter, req *http.Request) {
	orders, err := h.orderService.GetUserOrders(req.Context())
	if err != nil {
		h.log.Infof("orderService.GetUserOrders fail: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	var ordersResp []OrderResp
	for _, v := range orders {
		ordersResp = append(ordersResp, OrderResp{
			Number:     v.OrderNumber,
			Status:     v.Status,
			Accrual:    v.Accrual,
			UploadedAt: v.UploadedAt.Time.Format(time.RFC3339),
		})
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(res)
	if err := enc.Encode(ordersResp); err != nil {
		h.log.Infof("encode response fail: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetUserBalance(res http.ResponseWriter, req *http.Request) {
	balance, err := h.balanceService.GetUserBalance(req.Context())
	if err != nil {
		h.log.Infof("balanceService.GetUserBalance fail: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := BalanceResp{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(res)
	if err := enc.Encode(resp); err != nil {
		h.log.Infof("encode response fail: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetUserBalanceWithdraw(res http.ResponseWriter, req *http.Request) {
	withdrawals, err := h.withdrawalService.GetUserWithdrawals(req.Context())
	if err != nil {
		h.log.Infof("withdrawalService.GetUserWithdrawals: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	var withdrawalsResp []WithdrawalResp
	for _, v := range withdrawals {
		withdrawalsResp = append(withdrawalsResp, WithdrawalResp{
			Order:       v.OrderNumber,
			Sum:         v.Sum,
			ProcessedAt: v.ProcessedAt.Time.Format(time.RFC3339),
		})
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(res)
	if err := enc.Encode(withdrawalsResp); err != nil {
		h.log.Infof("encode response fail: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PostUserBalanceWithdraw(res http.ResponseWriter, req *http.Request) {
	var reqStr struct {
		Order string `json:"order"`
		Sum   int    `json:"sum"`
	}

	defer req.Body.Close()
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqStr); err != nil {
		h.log.Infof("decode request body: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if reqStr.Order == "" || reqStr.Sum == 0 {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if !h.isValidLuhn(reqStr.Order) {
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	err := h.withdrawalService.SaveWithdrawal(req.Context(), reqStr.Order, reqStr.Sum)
	if err != nil {
		if errors.Is(err, withdrawal.ErrNoMoney) {
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		h.log.Infof("withdrawalService.SaveWithdrawal: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (h *Handler) isValidLuhn(number string) bool {
	// Проверяем что строка состоит только из цифр
	for _, r := range number {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	// Алгоритм Луна требует как минимум 2 цифры
	if len(number) < 2 {
		return false
	}

	sum := 0
	// Идем по цифрам справа налево
	for i := 0; i < len(number); i++ {
		digit, _ := strconv.Atoi(string(number[len(number)-1-i]))

		// Каждую вторую цифру умножаем на 2
		if i%2 == 1 {
			digit *= 2
			if digit > 9 {
				digit = digit%10 + digit/10
			}
		}
		sum += digit
	}

	// Сумма должна быть кратна 10
	return sum%10 == 0
}
