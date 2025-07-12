package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal"

	withdrawalService "github.com/eduardtungatarov/gofermart/internal/service/withdrawal"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal/queries"

	middlewareMocks "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"
	withdrawalMocks "github.com/eduardtungatarov/gofermart/internal/service/withdrawal/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/eduardtungatarov/gofermart/internal/middleware"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/handlers"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

func TestPostUserBalanceWithdrawEndpoint(t *testing.T) {
	type reqStr struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}

	tests := []struct {
		name               string
		requestBody        reqStr
		requestContentType string
		mockSetup          func(m *withdrawalMocks.WithdrawalRepository)
		expectedHTTPStatus int
	}{
		{
			name: "success_auth",
			requestBody: reqStr{
				Order: "2377225624",
				Sum:   751,
			},
			requestContentType: "application/json",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				m.On("SaveWithdrawal", mock.Anything, queries.Withdrawal{
					UserID:      1,
					OrderNumber: "2377225624",
					Sum:         75100,
				}).Return(nil)
			},
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name: "incorrect_content_type",
			requestBody: reqStr{
				Order: "2377225624",
				Sum:   751,
			},
			requestContentType: "text/plain",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				//
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "miss_order",
			requestBody: reqStr{
				Sum: 751,
			},
			requestContentType: "text/plain",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				//
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "incorrect_order_number",
			requestBody: reqStr{
				Order: "123",
				Sum:   751,
			},
			requestContentType: "application/json",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
			},
			expectedHTTPStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "no_money_bro",
			requestBody: reqStr{
				Order: "2377225624",
				Sum:   751,
			},
			requestContentType: "application/json",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				m.On("SaveWithdrawal", mock.Anything, queries.Withdrawal{
					UserID:      1,
					OrderNumber: "2377225624",
					Sum:         75100,
				}).Return(withdrawal.ErrNoMoney)
			},
			expectedHTTPStatus: http.StatusPaymentRequired,
		},
		{
			name: "db_error",
			requestBody: reqStr{
				Order: "2377225624",
				Sum:   751,
			},
			requestContentType: "application/json",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				m.On("SaveWithdrawal", mock.Anything, queries.Withdrawal{
					UserID:      1,
					OrderNumber: "2377225624",
					Sum:         75100,
				}).Return(errors.New("db error"))
			},
			expectedHTTPStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		// Настраиваем сервер.
		wRepo := withdrawalMocks.NewWithdrawalRepository(t)
		tt.mockSetup(wRepo)
		authSrv := middlewareMocks.NewAuthService(t)
		authSrv.On("GetUserIDByToken", mock.Anything).
			Return(1, nil)
		m := middleware.MakeMiddleware(
			zap.NewNop().Sugar(),
			authSrv,
		)
		h := handlers.MakeHandler(
			zap.NewNop().Sugar(),
			nil,
			nil,
			nil,
			withdrawalService.New(wRepo),
		)
		srv := NewServer(
			config.Config{},
			h,
			m,
		)

		// Строю запрос.
		body, err := json.Marshal(tt.requestBody)
		assert.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", tt.requestContentType)
		req.Header.Set("Authorization", "Bearer token")

		// Выполняю запрос.
		rec := httptest.NewRecorder()
		srv.GetRouter().ServeHTTP(rec, req)

		// Проверяю ответ.
		assert.Equal(t, tt.expectedHTTPStatus, rec.Code, "want http status = %v, got = %v", tt.expectedHTTPStatus, rec.Code)

		// Проверяю вызовы мока.
		wRepo.AssertExpectations(t)
	}
}
