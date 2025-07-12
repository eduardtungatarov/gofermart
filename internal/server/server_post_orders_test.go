package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eduardtungatarov/gofermart/internal/repository/order"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"

	orderService "github.com/eduardtungatarov/gofermart/internal/service/order"

	middlewareMocks "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"
	orderMocks "github.com/eduardtungatarov/gofermart/internal/service/order/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/eduardtungatarov/gofermart/internal/middleware"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/handlers"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

func TestPostOrdersEndpoint(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        string
		requestContentType string
		mockSetup          func(m *orderMocks.OrderRepository)
		expectedHTTPStatus int
	}{
		{
			name:               "success_auth",
			requestBody:        "12345678903", // valid order number
			requestContentType: "text/plain",
			mockSetup: func(m *orderMocks.OrderRepository) {
				m.On("SaveOrder", mock.Anything, queries.Order{
					UserID:      1,
					OrderNumber: "12345678903",
					Status:      "NEW",
					Accrual:     0,
				}).Return(queries.Order{}, nil)
			},
			expectedHTTPStatus: http.StatusAccepted,
		},
		{
			name:               "invalid_order_number",
			requestBody:        "1233", // ivalid order number
			requestContentType: "text/plain",
			mockSetup: func(m *orderMocks.OrderRepository) {
				//
			},
			expectedHTTPStatus: http.StatusUnprocessableEntity,
		},
		{
			name:               "empty_order_number",
			requestBody:        "",
			requestContentType: "text/plain",
			mockSetup: func(m *orderMocks.OrderRepository) {
				//
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name:               "incorrect_content_type",
			requestBody:        "",
			requestContentType: "application/json", // incorrect
			mockSetup: func(m *orderMocks.OrderRepository) {
				//
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name:               "order_already_upload_by_user",
			requestBody:        "12345678903", // valid order number
			requestContentType: "text/plain",
			mockSetup: func(m *orderMocks.OrderRepository) {
				m.On("SaveOrder", mock.Anything, queries.Order{
					UserID:      1,
					OrderNumber: "12345678903",
					Status:      "NEW",
					Accrual:     0,
				}).Return(queries.Order{}, order.ErrOrderAlreadyExists)
				// Заказ принадлежил юзеру с ID = 1;
				m.On("FindOrderByOrderNumber", mock.Anything, "12345678903").
					Return(queries.Order{UserID: 1}, nil)
			},
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name:               "order_already_upload_by_another_user",
			requestBody:        "12345678903", // valid order number
			requestContentType: "text/plain",
			mockSetup: func(m *orderMocks.OrderRepository) {
				m.On("SaveOrder", mock.Anything, queries.Order{
					UserID:      1,
					OrderNumber: "12345678903",
					Status:      "NEW",
					Accrual:     0,
				}).Return(queries.Order{}, order.ErrOrderAlreadyExists)
				// Заказ принадлежил юзеру с ID = 2;
				m.On("FindOrderByOrderNumber", mock.Anything, "12345678903").
					Return(queries.Order{UserID: 2}, nil)
			},
			expectedHTTPStatus: http.StatusConflict,
		},
		{
			name:               "db_error",
			requestBody:        "12345678903", // valid order number
			requestContentType: "text/plain",
			mockSetup: func(m *orderMocks.OrderRepository) {
				m.On("SaveOrder", mock.Anything, queries.Order{
					UserID:      1,
					OrderNumber: "12345678903",
					Status:      "NEW",
					Accrual:     0,
				}).Return(queries.Order{}, errors.New("db error"))
			},
			expectedHTTPStatus: http.StatusInternalServerError,
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
		req, err := http.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader(tt.requestBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", tt.requestContentType)
		req.Header.Set("Authorization", "Bearer token")

		// Выполняю запрос.
		rec := httptest.NewRecorder()
		srv.GetRouter().ServeHTTP(rec, req)

		// Проверяю ответ.
		assert.Equal(t, tt.expectedHTTPStatus, rec.Code, "want http status = %v, got = %v", tt.expectedHTTPStatus, rec.Code)

		// Проверяю вызовы мока.
		orderRepo.AssertExpectations(t)
	}
}
