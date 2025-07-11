package server

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orderService "github.com/eduardtungatarov/gofermart/internal/service/order"
	orderMocks "github.com/eduardtungatarov/gofermart/internal/service/order/mocks"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"

	middlewareMocks "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/eduardtungatarov/gofermart/internal/middleware"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/handlers"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

func TestGetUserOrdersEndpoint(t *testing.T) {
	tests := []struct {
		name                string
		mockSetup           func(m *orderMocks.OrderRepository)
		expectedHTTPStatus  int
		expectedContentType string
		expectedBody        string
	}{
		{
			name: "success",
			mockSetup: func(m *orderMocks.OrderRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return([]queries.Order{
					{
						OrderNumber: "9278923470",
						Status:      "PROCESSED",
						Accrual:     50000,
						UploadedAt:  sql.NullTime{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
					},
					{
						OrderNumber: "346436439",
						Status:      "INVALID",
						UploadedAt:  sql.NullTime{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
					},
				}, nil)
			},
			expectedHTTPStatus:  http.StatusOK,
			expectedContentType: "application/json",
			expectedBody:        `[{"number":"9278923470","status":"PROCESSED","accrual":500,"uploaded_at":"2023-01-01T00:00:00Z"},{"number":"346436439","status":"INVALID","uploaded_at":"2023-01-01T00:00:00Z"}]`,
		},
		{
			name: "empty_order",
			mockSetup: func(m *orderMocks.OrderRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return([]queries.Order{}, nil)
			},
			expectedHTTPStatus:  http.StatusNoContent,
			expectedContentType: "",
			expectedBody:        ``,
		},
		{
			name: "internal_err",
			mockSetup: func(m *orderMocks.OrderRepository) {
				m.On("FindByUserID", mock.Anything, 1).Return([]queries.Order{}, errors.New("db err"))
			},
			expectedHTTPStatus:  http.StatusInternalServerError,
			expectedContentType: "",
			expectedBody:        ``,
		},
	}

	for _, tt := range tests {
		// Настраиваем сервер.
		orderRepo := orderMocks.NewOrderRepository(t)
		tt.mockSetup(orderRepo)
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
			orderService.New(orderRepo),
			nil,
			nil,
		)
		srv := NewServer(
			config.Config{},
			h,
			m,
		)

		// Строю запрос.
		req, err := http.NewRequest(http.MethodGet, "/api/user/orders", nil)
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
		orderRepo.AssertExpectations(t)
	}
}
