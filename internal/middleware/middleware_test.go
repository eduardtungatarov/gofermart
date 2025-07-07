package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/middleware/mocks"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWithAuth(t *testing.T) {
	testCases := []struct {
		name           string
		setupMock      func(*mocks.AuthService)
		authHeader     string
		expectedStatus int
		expectUserID   bool
		expectedUserID int
	}{
		{
			name: "Successful_authentication_with_valid_token",
			setupMock: func(m *mocks.AuthService) {
				m.On("GetUserIDByToken", "valid.token.123").Return(123, nil)
			},
			authHeader:     "Bearer valid.token.123",
			expectedStatus: http.StatusOK,
			expectUserID:   true,
			expectedUserID: 123,
		},
		{
			name:           "Missing_authorization_header",
			setupMock:      func(m *mocks.AuthService) {},
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name: "Incorrect_token",
			setupMock: func(m *mocks.AuthService) {
				m.On("GetUserIDByToken", "expired.token").Return(0, jwt.ErrTokenExpired)
			},
			authHeader:     "Bearer expired.token",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := mocks.NewAuthService(t)
			tc.setupMock(mockAuthService)

			mw := middleware.MakeMiddleware(zap.NewNop().Sugar(), mockAuthService)

			handlerCalled := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				if tc.expectUserID {
					userID := r.Context().Value(config.UserIDKeyName)
					assert.Equal(t, tc.expectedUserID, userID, "UserID in context mismatch")
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			rr := httptest.NewRecorder()
			mw.WithAuth(handler).ServeHTTP(rr, req)

			// Assertions
			assert.Equal(t, tc.expectedStatus, rr.Code, "HTTP status code mismatch")
			if tc.expectedStatus == http.StatusOK {
				assert.True(t, handlerCalled, "Handler should be called")
			} else {
				assert.False(t, handlerCalled, "Handler should not be called")
			}
			mockAuthService.AssertExpectations(t)
		})
	}
}
