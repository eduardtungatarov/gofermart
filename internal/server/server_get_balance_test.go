package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardtungatarov/gofermart/internal/repository"

	balanceService "github.com/eduardtungatarov/gofermart/internal/service/balance"

	"github.com/eduardtungatarov/gofermart/internal/repository/balance/queries"

	middlewareMocks "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"
	balanceMocks "github.com/eduardtungatarov/gofermart/internal/service/balance/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/eduardtungatarov/gofermart/internal/middleware"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/handlers"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

func TestGetBalanceEndpoint(t *testing.T) {
	tests := []struct {
		name                string
		mockSetup           func(m *balanceMocks.BalanceRepository)
		expectedHTTPStatus  int
		expectedContentType string
		expectedBody        string
	}{
		{
			name: "success",
			mockSetup: func(m *balanceMocks.BalanceRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return(queries.Balance{
					Current:   10000,
					Withdrawn: 5000,
				}, nil)
			},
			expectedHTTPStatus:  http.StatusOK,
			expectedContentType: "application/json",
			expectedBody:        `{"current":100,"withdrawn":50}`,
		},
		{
			name: "correct_response_when_no_balance_record",
			mockSetup: func(m *balanceMocks.BalanceRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return(queries.Balance{}, repository.ErrNoModel)
			},
			expectedHTTPStatus:  http.StatusOK,
			expectedContentType: "application/json",
			expectedBody:        `{"current":0,"withdrawn":0}`,
		},
		{
			name: "internal_err",
			mockSetup: func(m *balanceMocks.BalanceRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return(queries.Balance{}, errors.New("db err"))
			},
			expectedHTTPStatus:  http.StatusInternalServerError,
			expectedContentType: "",
			expectedBody:        ``,
		},
	}

	for _, tt := range tests {
		// Настраиваем сервер.
		balanceRepo := balanceMocks.NewBalanceRepository(t)
		tt.mockSetup(balanceRepo)
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
			balanceService.New(balanceRepo),
			nil,
		)
		srv := NewServer(
			config.Config{},
			h,
			m,
		)

		// Строю запрос.
		req, err := http.NewRequest(http.MethodGet, "/api/user/balance", nil)
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
		balanceRepo.AssertExpectations(t)
	}
}
