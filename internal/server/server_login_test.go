package server_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eduardtungatarov/gofermart/internal/service/auth/mocks"
	orderMocks "github.com/eduardtungatarov/gofermart/internal/service/order/mocks"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"
	"github.com/eduardtungatarov/gofermart/internal/server"
	"github.com/eduardtungatarov/gofermart/internal/service/auth"
	"github.com/eduardtungatarov/gofermart/internal/service/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestLoginEndpoint(t *testing.T) {
	// Тестовые данные
	validUser := queries.User{
		Login:    "valid_user",
		Password: `$2a$14$aC862HErmLXW7cwC/17IaeeJpVuJhQDNBbVmMWMvip5MVixCd/Tnu`, // bcrypt хэш для "valid_password"
	}

	tests := []struct {
		name           string
		request        map[string]string
		mockBehavior   func(*mocks.UserRepository)
		expectedStatus int
		wantToken      bool
	}{
		{
			name: "successful_login",
			request: map[string]string{
				"login":    "valid_user",
				"password": "valid_password",
			},
			mockBehavior: func(m *mocks.UserRepository) {
				m.On("FindUserByLogin", mock.Anything, "valid_user").Return(validUser, nil)
			},
			expectedStatus: http.StatusOK,
			wantToken:      true,
		},
		{
			name: "wrong_password",
			request: map[string]string{
				"login":    "valid_user",
				"password": "wrong_password",
			},
			mockBehavior: func(m *mocks.UserRepository) {
				m.On("FindUserByLogin", mock.Anything, "valid_user").Return(validUser, nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user_not_found",
			request: map[string]string{
				"login":    "unknown_user",
				"password": "any_password",
			},
			mockBehavior: func(m *mocks.UserRepository) {
				m.On("FindUserByLogin", mock.Anything, "unknown_user").
					Return(queries.User{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "empty_login",
			request: map[string]string{
				"login":    "",
				"password": "any_password",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid_request_format",
			request: map[string]string{
				"username": "valid_user", // неправильное поле
				"password": "valid_password",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Инициализация моков
			repo := new(mocks.UserRepository)
			orderRepo := orderMocks.NewOrderRepository(t)
			if tt.mockBehavior != nil {
				tt.mockBehavior(repo)
			}

			// 2. Создание тестового сервера
			logger := zap.NewNop().Sugar()
			authService := auth.New(repo)
			orderService := order.New(orderRepo)

			handler := handlers.MakeHandler(logger, authService, orderService)
			srv := server.NewServer(config.Config{}, handler, middleware.MakeMiddleware(logger, authService))

			// 3. Формирование запроса
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// 4. Выполнение запроса
			rec := httptest.NewRecorder()
			srv.GetRouter().ServeHTTP(rec, req)

			// 5. Проверки
			assert.Equal(t, tt.expectedStatus, rec.Code, "неверный статус код")

			if tt.wantToken {
				assert.NotEmpty(t, rec.Header().Get("Authorization"), "токен не должен быть пустым")
			}

			// 6. Проверка вызовов моков
			repo.AssertExpectations(t)
		})
	}
}
