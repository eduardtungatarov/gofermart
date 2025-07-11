package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	userRepository "github.com/eduardtungatarov/gofermart/internal/repository/user"

	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"

	authService "github.com/eduardtungatarov/gofermart/internal/service/auth"

	authMocks "github.com/eduardtungatarov/gofermart/internal/service/auth/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/eduardtungatarov/gofermart/internal/middleware"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/handlers"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

func TestUserRegisterEndpoint(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        map[string]string
		requestContentType string
		mockSetup          func(m *authMocks.UserRepository)
		expectedAuthHeader bool
		expectedHTTPStatus int
	}{
		{
			name: "success_registration",
			requestBody: map[string]string{
				"login":    "user",
				"password": "pwd",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				m.On("SaveUser", mock.Anything, mock.MatchedBy(func(user queries.User) bool {
					return user.Login == "user"
				})).Return(queries.User{ID: 1}, nil)
			},
			expectedAuthHeader: true,
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name:               "with_incorrect_content_type",
			requestBody:        map[string]string{},
			requestContentType: "text/plain", // incorrect
			mockSetup:          func(m *authMocks.UserRepository) {},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "miss_password",
			requestBody: map[string]string{
				"login":    "user",
				"password": "",
			},
			requestContentType: "application/json",
			mockSetup:          func(m *authMocks.UserRepository) {},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "miss_user",
			requestBody: map[string]string{
				"password": "pwd",
			},
			requestContentType: "application/json",
			mockSetup:          func(m *authMocks.UserRepository) {},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "user_already_exist",
			requestBody: map[string]string{
				"login":    "user",
				"password": "pwd",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				m.On("SaveUser", mock.Anything, mock.MatchedBy(func(user queries.User) bool {
					return user.Login == "user"
				})).Return(queries.User{}, userRepository.ErrUserAlreadyExists)
			},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusConflict,
		},
		{
			name: "db_error",
			requestBody: map[string]string{
				"login":    "user",
				"password": "pwd",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				m.On("SaveUser", mock.Anything, mock.MatchedBy(func(user queries.User) bool {
					return user.Login == "user"
				})).Return(queries.User{}, errors.New("db error"))
			},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		// Настраиваем сервер.
		userRepo := authMocks.NewUserRepository(t)
		tt.mockSetup(userRepo)
		m := middleware.MakeMiddleware(
			zap.NewNop().Sugar(),
			nil,
		)
		h := handlers.MakeHandler(
			zap.NewNop().Sugar(),
			authService.New(userRepo),
			nil,
			nil,
			nil,
		)
		srv := NewServer(
			config.Config{},
			h,
			m,
		)

		// Строю запрос.
		body, err := json.Marshal(tt.requestBody)
		assert.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", tt.requestContentType)

		// Выполняю запрос.
		rec := httptest.NewRecorder()
		srv.GetRouter().ServeHTTP(rec, req)

		// Проверяю ответ.
		if tt.expectedAuthHeader {
			assert.NotEmpty(t, tt.expectedAuthHeader, rec.Header().Get("Authorization"), "token must not be empty in the Authorization header")
		}
		assert.Equal(t, tt.expectedHTTPStatus, rec.Code, "want http status = %v, got = %v", tt.expectedHTTPStatus, rec.Code)

		// Проверяю вызовы мока.
		userRepo.AssertExpectations(t)
	}
}
