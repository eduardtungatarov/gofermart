package server_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eduardtungatarov/gofermart/internal/service/auth/mocks"
	orderMocks "github.com/eduardtungatarov/gofermart/internal/service/order/mocks"

	"github.com/eduardtungatarov/gofermart/internal/repository/user"

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

func TestRegisterEndpoint(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := config.Config{RunADDR: ":0"}

	tests := []struct {
		name           string
		setupMock      func(*mocks.UserRepository)
		requestBody    map[string]string
		contentType    string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
		checkMock      func(*testing.T, *mocks.UserRepository)
	}{
		{
			name: "successful_registration",
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
					Return(queries.User{ID: 1}, nil)
			},
			requestBody: map[string]string{
				"login":    "testuser",
				"password": "testpass",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				assert.NotEmpty(t, resp.Header.Get("Authorization"), "токен не должен быть пустым")
			},
			checkMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.AssertExpectations(t)
			},
		},
		{
			name: "user_already_exists",
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
					Return(queries.User{}, user.ErrUserAlreadyExists)
			},
			requestBody: map[string]string{
				"login":    "existinguser",
				"password": "testpass",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusConflict,
			checkMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.AssertExpectations(t)
			},
		},
		{
			name:      "invalid_request_missing_fields",
			setupMock: func(repo *mocks.UserRepository) {},
			requestBody: map[string]string{
				"login": "testuser",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			checkMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.AssertNotCalled(t, "SaveUser")
			},
		},
		{
			name:      "invalid_request_wrong_content_type",
			setupMock: func(repo *mocks.UserRepository) {},
			requestBody: map[string]string{
				"login":    "testuser",
				"password": "testpass",
			},
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
			checkMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.AssertNotCalled(t, "SaveUser")
			},
		},
		{
			name: "internal_server_error",
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
					Return(queries.User{}, errors.New("database error"))
			},
			requestBody: map[string]string{
				"login":    "testuser",
				"password": "testpass",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusInternalServerError,
			checkMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем мок репозитория
			userRepo := mocks.NewUserRepository(t)
			orderRepo := orderMocks.NewOrderRepository(t)
			tt.setupMock(userRepo)

			// Собираем сервисы
			authSrv := auth.New(userRepo)
			orderSrv := order.New(orderRepo)
			h := handlers.MakeHandler(logger, authSrv, orderSrv)
			m := middleware.MakeMiddleware(logger, authSrv)
			s := server.NewServer(cfg, h, m)

			// Создаем тестовый сервер
			testServer := httptest.NewServer(s.GetRouter())
			defer testServer.Close()

			// Подготовка запроса
			jsonBody, _ := json.Marshal(tt.requestBody)

			// Выполняем запрос
			resp, err := http.Post(
				testServer.URL+"/api/user/register",
				tt.contentType,
				bytes.NewBuffer(jsonBody),
			)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Проверяем результаты
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}

			if tt.checkMock != nil {
				tt.checkMock(t, userRepo)
			}
		})
	}
}

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
