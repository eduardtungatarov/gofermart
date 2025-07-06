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

	"github.com/eduardtungatarov/gofermart/internal/repository/user"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"
	"github.com/eduardtungatarov/gofermart/internal/server"
	"github.com/eduardtungatarov/gofermart/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestRegisterEndpoint(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := config.Config{RunADDR: ":0"}

	t.Run("successful_registration", func(t *testing.T) {
		// Настраиваем мок репозитория
		userRepo := mocks.NewUserRepository(t)
		userRepo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
			Return(queries.User{ID: 1}, nil)

		// Собираем сервисы
		authSrv := auth.New(userRepo)
		h := handlers.MakeHandler(logger, authSrv)
		m := middleware.MakeMiddleware(logger, authSrv)
		s := server.NewServer(cfg, h, m)

		// Создаем тестовый сервер
		testServer := httptest.NewServer(s.GetRouter())
		defer testServer.Close()

		// Подготовка запроса
		requestBody := map[string]string{
			"login":    "testuser",
			"password": "testpass",
		}
		jsonBody, _ := json.Marshal(requestBody)

		// Выполняем запрос
		resp, err := http.Post(
			testServer.URL+"/api/user/register",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Проверяем результаты
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("Authorization"), "токен не должен быть пустым")
		userRepo.AssertExpectations(t)
	})

	t.Run("user_already_exists", func(t *testing.T) {
		userRepo := mocks.NewUserRepository(t)
		userRepo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
			Return(queries.User{}, user.ErrUserAlreadyExists)

		authSrv := auth.New(userRepo)
		h := handlers.MakeHandler(logger, authSrv)
		m := middleware.MakeMiddleware(logger, authSrv)
		s := server.NewServer(cfg, h, m)

		testServer := httptest.NewServer(s.GetRouter())
		defer testServer.Close()

		requestBody := map[string]string{
			"login":    "existinguser",
			"password": "testpass",
		}
		jsonBody, _ := json.Marshal(requestBody)

		resp, err := http.Post(
			testServer.URL+"/api/user/register",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		userRepo.AssertExpectations(t)
	})

	t.Run("invalid_request_missing_fields", func(t *testing.T) {
		userRepo := mocks.NewUserRepository(t)
		authSrv := auth.New(userRepo)
		h := handlers.MakeHandler(logger, authSrv)
		m := middleware.MakeMiddleware(logger, authSrv)
		s := server.NewServer(cfg, h, m)

		testServer := httptest.NewServer(s.GetRouter())
		defer testServer.Close()

		// Тело запроса без поля password
		requestBody := map[string]string{
			"login": "testuser",
		}
		jsonBody, _ := json.Marshal(requestBody)

		resp, err := http.Post(
			testServer.URL+"/api/user/register",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		userRepo.AssertNotCalled(t, "SaveUser")
	})

	t.Run("invalid_request_wrong_content_type", func(t *testing.T) {
		userRepo := mocks.NewUserRepository(t)
		authSrv := auth.New(userRepo)
		h := handlers.MakeHandler(logger, authSrv)
		m := middleware.MakeMiddleware(logger, authSrv)
		s := server.NewServer(cfg, h, m)

		testServer := httptest.NewServer(s.GetRouter())
		defer testServer.Close()

		requestBody := map[string]string{
			"login":    "testuser",
			"password": "testpass",
		}
		jsonBody, _ := json.Marshal(requestBody)

		// Отправляем с неправильным Content-Type
		resp, err := http.Post(
			testServer.URL+"/api/user/register",
			"text/plain",
			bytes.NewBuffer(jsonBody),
		)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		userRepo.AssertNotCalled(t, "SaveUser")
	})

	t.Run("internal_server_error", func(t *testing.T) {
		userRepo := mocks.NewUserRepository(t)
		userRepo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
			Return(queries.User{}, errors.New("database error"))

		authSrv := auth.New(userRepo)
		h := handlers.MakeHandler(logger, authSrv)
		m := middleware.MakeMiddleware(logger, authSrv)
		s := server.NewServer(cfg, h, m)

		testServer := httptest.NewServer(s.GetRouter())
		defer testServer.Close()

		requestBody := map[string]string{
			"login":    "testuser",
			"password": "testpass",
		}
		jsonBody, _ := json.Marshal(requestBody)

		resp, err := http.Post(
			testServer.URL+"/api/user/register",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		userRepo.AssertExpectations(t)
	})
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
			if tt.mockBehavior != nil {
				tt.mockBehavior(repo)
			}

			// 2. Создание тестового сервера
			logger := zap.NewNop().Sugar()
			authService := auth.New(repo)
			handler := handlers.MakeHandler(logger, authService)
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
