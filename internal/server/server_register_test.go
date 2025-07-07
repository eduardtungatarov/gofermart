package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authMocks "github.com/eduardtungatarov/gofermart/internal/service/auth/mocks"
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
		setupMock      func(*authMocks.UserRepository)
		requestBody    map[string]string
		contentType    string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
		checkMock      func(*testing.T, *authMocks.UserRepository)
	}{
		{
			name: "successful_registration",
			setupMock: func(repo *authMocks.UserRepository) {
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
			checkMock: func(t *testing.T, repo *authMocks.UserRepository) {
				repo.AssertExpectations(t)
			},
		},
		{
			name: "user_already_exists",
			setupMock: func(repo *authMocks.UserRepository) {
				repo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
					Return(queries.User{}, user.ErrUserAlreadyExists)
			},
			requestBody: map[string]string{
				"login":    "existinguser",
				"password": "testpass",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusConflict,
			checkMock: func(t *testing.T, repo *authMocks.UserRepository) {
				repo.AssertExpectations(t)
			},
		},
		{
			name:      "invalid_request_missing_fields",
			setupMock: func(repo *authMocks.UserRepository) {},
			requestBody: map[string]string{
				"login": "testuser",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			checkMock: func(t *testing.T, repo *authMocks.UserRepository) {
				repo.AssertNotCalled(t, "SaveUser")
			},
		},
		{
			name:      "invalid_request_wrong_content_type",
			setupMock: func(repo *authMocks.UserRepository) {},
			requestBody: map[string]string{
				"login":    "testuser",
				"password": "testpass",
			},
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
			checkMock: func(t *testing.T, repo *authMocks.UserRepository) {
				repo.AssertNotCalled(t, "SaveUser")
			},
		},
		{
			name: "internal_server_error",
			setupMock: func(repo *authMocks.UserRepository) {
				repo.On("SaveUser", mock.Anything, mock.AnythingOfType("queries.User")).
					Return(queries.User{}, errors.New("database error"))
			},
			requestBody: map[string]string{
				"login":    "testuser",
				"password": "testpass",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusInternalServerError,
			checkMock: func(t *testing.T, repo *authMocks.UserRepository) {
				repo.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем мок репозитория
			userRepo := authMocks.NewUserRepository(t)
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
