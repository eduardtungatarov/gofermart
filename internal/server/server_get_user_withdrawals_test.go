package server

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestGetUserWithdrawalsEndpoint(t *testing.T) {
	tests := []struct {
		name                string
		mockSetup           func(m *withdrawalMocks.WithdrawalRepository)
		expectedHTTPStatus  int
		expectedContentType string
		expectedBody        string
	}{
		{
			name: "success",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return([]queries.Withdrawal{
					{
						OrderNumber: "2377225624",
						Sum:         50000,
						ProcessedAt: sql.NullTime{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
					},
				}, nil)
			},
			expectedHTTPStatus:  http.StatusOK,
			expectedContentType: "application/json",
			expectedBody:        `[{"order":"2377225624", "processed_at":"2023-01-01T00:00:00Z", "sum":500}]`,
		},
		{
			name: "empty_withdrawal",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return([]queries.Withdrawal{}, nil)
			},
			expectedHTTPStatus:  http.StatusNoContent,
			expectedContentType: "",
			expectedBody:        ``,
		},
		{
			name: "internal_err",
			mockSetup: func(m *withdrawalMocks.WithdrawalRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return([]queries.Withdrawal{}, errors.New("db err"))
			},
			expectedHTTPStatus:  http.StatusInternalServerError,
			expectedContentType: "",
			expectedBody:        ``,
		},
	}

	for _, tt := range tests {
		// Настраиваем сервер.
		withdrawalRepo := withdrawalMocks.NewWithdrawalRepository(t)
		tt.mockSetup(withdrawalRepo)
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
			withdrawalService.New(withdrawalRepo),
		)
		srv := NewServer(
			config.Config{},
			h,
			m,
		)

		// Строю запрос.
		req, err := http.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")

		// Выполняю запрос.
		rec := httptest.NewRecorder()
		srv.GetRouter().ServeHTTP(rec, req)

		// Проверяю ответ.
		assert.Equal(t, tt.expectedHTTPStatus, rec.Code, "want http status = %v, got = %v", tt.expectedHTTPStatus, rec.Code)
		if tt.expectedBody != "" {
			assert.JSONEq(t, tt.expectedBody, rec.Body.String(), "want body = %v, got = %v", tt.expectedBody, rec.Body.String())
		}
		if tt.expectedContentType != "" {
			assert.Equal(t, tt.expectedContentType, "application/json", "content-type response is wrong, must be json")
		}

		// Проверяю вызовы мока.
		withdrawalRepo.AssertExpectations(t)
	}
}
