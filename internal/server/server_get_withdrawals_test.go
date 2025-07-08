package server_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mocksMiddleware "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"

	"github.com/eduardtungatarov/gofermart/internal/handlers/mocks"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal/queries"
	"github.com/eduardtungatarov/gofermart/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestGetUserBalanceWithdraw(t *testing.T) {
	tests := []struct {
		name             string
		mockWithdrawals  []queries.Withdrawal
		mockError        error
		expectedStatus   int
		expectedResponse []handlers.WithdrawalResp
	}{
		{
			name: "successful_request_with_withdrawals",
			mockWithdrawals: []queries.Withdrawal{
				{
					OrderNumber: "123456",
					Sum:         100,
					ProcessedAt: sql.NullTime{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
				},
				{
					OrderNumber: "789012",
					Sum:         200,
					ProcessedAt: sql.NullTime{Time: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), Valid: true},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedResponse: []handlers.WithdrawalResp{
				{
					Order:       "123456",
					Sum:         100,
					ProcessedAt: "2023-01-01T00:00:00Z",
				},
				{
					Order:       "789012",
					Sum:         200,
					ProcessedAt: "2023-01-02T00:00:00Z",
				},
			},
		},
		{
			name:             "no_withdrawals_found",
			mockWithdrawals:  []queries.Withdrawal{},
			mockError:        nil,
			expectedStatus:   http.StatusNoContent,
			expectedResponse: nil,
		},
		{
			name:             "internal_server_error",
			mockWithdrawals:  nil,
			mockError:        assert.AnError,
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWithdrawalService := mocks.NewWithdrawalService(t)
			mockWithdrawalService.On("GetUserWithdrawals", mock.Anything).
				Return(tt.mockWithdrawals, tt.mockError)

			h := handlers.MakeHandler(
				zap.NewNop().Sugar(),
				nil,
				nil,
				nil,
				mockWithdrawalService,
			)

			authSrv := mocksMiddleware.NewAuthService(t)
			authSrv.On("GetUserIDByToken", mock.Anything).
				Return(1, nil)

			srv := server.NewServer(
				config.Config{},
				h,
				middleware.MakeMiddleware(zap.NewNop().Sugar(), authSrv),
			)

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance/withdraw", nil)
			req.Header.Set("Authorization", "Bearer blatest")
			rr := httptest.NewRecorder()

			router := srv.GetRouter()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedResponse != nil {
				var response []handlers.WithdrawalResp
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, response)
			}

			mockWithdrawalService.AssertExpectations(t)
		})
	}
}
