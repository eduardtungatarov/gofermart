package server_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eduardtungatarov/gofermart/internal/handlers/mocks"
	mocksMiddleware "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"
	"github.com/eduardtungatarov/gofermart/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestGetUserOrders(t *testing.T) {
	tests := []struct {
		name           string
		mockOrders     []queries.Order
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful_get_orders",
			mockOrders: []queries.Order{
				{
					OrderNumber: "123",
					Status:      "PROCESSED",
					Accrual:     100,
					UploadedAt:  sql.NullTime{Time: time.Now(), Valid: true},
				},
				{
					OrderNumber: "456",
					Status:      "NEW",
					Accrual:     0,
					UploadedAt:  sql.NullTime{Time: time.Now(), Valid: true},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"number":"123","status":"PROCESSED","accrual":100,"uploaded_at":"` + time.Now().Format(time.RFC3339) + `"},{"number":"456","status":"NEW","accrual":0,"uploaded_at":"` + time.Now().Format(time.RFC3339) + `"}]`,
		},
		{
			name:           "no_orders",
			mockOrders:     []queries.Order{},
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name:           "internal_server_error",
			mockOrders:     nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем мок сервиса заказов
			mockOrderService := mocks.NewOrderService(t)
			mockOrderService.On("GetUserOrders", mock.Anything).
				Return(tt.mockOrders, tt.mockError)

			// Создаем обработчик
			handler := handlers.MakeHandler(
				zap.NewNop().Sugar(),
				nil, // authService не нужен для этого теста
				mockOrderService,
				nil, // balanceService не нужен для этого теста
			)

			// Создаем сервер
			authSrv := mocksMiddleware.NewAuthService(t)
			authSrv.On("GetUserIDByToken", mock.Anything).
				Return(1, nil)

			srv := server.NewServer(
				config.Config{},
				handler,
				middleware.MakeMiddleware(zap.NewNop().Sugar(), authSrv),
			)

			// Создаем тестовый запрос
			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			req.Header.Set("Authorization", "Bearer blatest")
			w := httptest.NewRecorder()

			// Выполняем запрос
			srv.GetRouter().ServeHTTP(w, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Проверяем тело ответа, если ожидается
			if tt.expectedBody != "" {
				var body []handlers.OrderResp
				err := json.Unmarshal(w.Body.Bytes(), &body)
				assert.NoError(t, err)

				var expectedBody []handlers.OrderResp
				err = json.Unmarshal([]byte(tt.expectedBody), &expectedBody)
				assert.NoError(t, err)

				assert.Equal(t, expectedBody, body)
			} else {
				assert.Empty(t, w.Body.String())
			}

			// Проверяем, что мок был вызван
			mockOrderService.AssertExpectations(t)
		})
	}
}
