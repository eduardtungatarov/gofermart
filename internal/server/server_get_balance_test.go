package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mocksMiddleware "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"

	"github.com/eduardtungatarov/gofermart/internal/handlers/mocks"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/repository/balance/queries"
	"github.com/eduardtungatarov/gofermart/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestServer_GetUserBalance(t *testing.T) {
	tests := []struct {
		name           string
		setupAuth      bool
		mockBalance    queries.Balance
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful_balance_retrieval",
			setupAuth:      true,
			mockBalance:    queries.Balance{Current: 105, Withdrawn: 20},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"current":105,"withdrawn":20}`,
		},
		{
			name:           "no_balance_record_found",
			setupAuth:      true,
			mockBalance:    queries.Balance{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"current":0,"withdrawn":0}`,
		},
		{
			name:           "internal_server_error",
			setupAuth:      true,
			mockBalance:    queries.Balance{},
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настройка моков
			balanceService := mocks.NewBalanceService(t)

			if tt.setupAuth {
				balanceService.On("GetUserBalance", mock.Anything).Return(tt.mockBalance, tt.mockError)
			}

			// Создание тестового обработчика
			h := handlers.MakeHandler(
				zap.NewNop().Sugar(),
				nil, // authService не нужен для этого теста
				nil, // orderService не нужен
				balanceService,
			)

			// Создание тестового сервера
			cfg := config.Config{}
			authSrv := mocksMiddleware.NewAuthService(t)
			authSrv.On("GetUserIDByToken", mock.Anything).
				Return(1, nil)
			m := middleware.MakeMiddleware(zap.NewNop().Sugar(), authSrv)
			srv := server.NewServer(cfg, h, m)

			// Создание тестового запроса
			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			req.Header.Set("Authorization", "Bearer blatest")
			w := httptest.NewRecorder()

			// Выполнение запроса
			srv.GetRouter().ServeHTTP(w, req)

			// Проверки
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			// Проверка вызовов моков
			if tt.setupAuth {
				balanceService.AssertExpectations(t)
			}
		})
	}
}
