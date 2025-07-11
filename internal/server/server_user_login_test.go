package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardtungatarov/gofermart/internal/repository"

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

func TestUserLoginEndpoint(t *testing.T) {
	validUser := queries.User{
		Login:    "valid_user",
		Password: `$2a$14$aC862HErmLXW7cwC/17IaeeJpVuJhQDNBbVmMWMvip5MVixCd/Tnu`, // bcrypt хэш для "valid_password"
	}

	tests := []struct {
		name               string
		requestBody        map[string]string
		requestContentType string
		mockSetup          func(m *authMocks.UserRepository)
		expectedAuthHeader bool
		expectedHTTPStatus int
	}{
		{
			name: "success_auth",
			requestBody: map[string]string{
				"login":    "valid_user",
				"password": "valid_password",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				m.On("FindUserByLogin", mock.Anything, "valid_user").Return(validUser, nil)
			},
			expectedAuthHeader: true,
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name: "fail_auth_wrong_password",
			requestBody: map[string]string{
				"login":    "valid_user",
				"password": "wrong_password",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				m.On("FindUserByLogin", mock.Anything, "valid_user").Return(validUser, nil)
			},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusUnauthorized,
		},
		{
			name: "fail_auth_user_not_found",
			requestBody: map[string]string{
				"login":    "unexist_user",
				"password": "password",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				m.On("FindUserByLogin", mock.Anything, "unexist_user").Return(queries.User{}, repository.ErrNoModel)
			},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusUnauthorized,
		},
		{
			name: "empty_login",
			requestBody: map[string]string{
				"login":    "",
				"password": "valid_password",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				//
			},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "empty_pwd",
			requestBody: map[string]string{
				"login":    "valid_user",
				"password": "",
			},
			requestContentType: "application/json",
			mockSetup: func(m *authMocks.UserRepository) {
				//
			},
			expectedAuthHeader: false,
			expectedHTTPStatus: http.StatusBadRequest,
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
		req, err := http.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
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
