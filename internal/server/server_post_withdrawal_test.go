package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal"

	mocksMiddleware "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"
	"github.com/stretchr/testify/mock"

	"github.com/eduardtungatarov/gofermart/internal/handlers/mocks"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPostUserBalanceWithdraw(t *testing.T) {
	type args struct {
		order string
		sum   int
		token string
	}
	type want struct {
		status int
	}
	type testCase struct {
		name string
		args args
		want want
		mock func(
			withdrawalService *mocks.WithdrawalService,
		)
	}

	tests := []testCase{
		{
			name: "successful_withdrawal",
			args: args{
				order: "12345678903", // valid Luhn number
				sum:   500,
			},
			want: want{
				status: http.StatusOK,
			},
			mock: func(
				withdrawalService *mocks.WithdrawalService,
			) {
				withdrawalService.On("SaveWithdrawal", mock.Anything, "12345678903", 500).
					Return(nil)
			},
		},
		{
			name: "invalid_Luhn_number",
			args: args{
				order: "12345678901", // invalid Luhn number
				sum:   500,
			},
			want: want{
				status: http.StatusUnprocessableEntity,
			},
			mock: func(
				withdrawalService *mocks.WithdrawalService,
			) {
				//
			},
		},
		{
			name: "insufficient_funds",
			args: args{
				order: "12345678903", // valid Luhn number
				sum:   500,
			},
			want: want{
				status: http.StatusUnprocessableEntity,
			},
			mock: func(
				withdrawalService *mocks.WithdrawalService,
			) {
				withdrawalService.On("SaveWithdrawal", mock.Anything, "12345678903", 500).
					Return(withdrawal.ErrNoMoney)
			},
		},
		{
			name: "invalid_request_body",
			args: args{
				order: "",
				sum:   0,
			},
			want: want{
				status: http.StatusBadRequest,
			},
			mock: func(
				withdrawalService *mocks.WithdrawalService,
			) {
				//
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withdrawalService := mocks.NewWithdrawalService(t)
			tt.mock(withdrawalService)

			logger := zap.NewNop().Sugar()
			h := handlers.MakeHandler(
				logger,
				nil,
				nil,
				nil,
				withdrawalService,
			)

			authSrv := mocksMiddleware.NewAuthService(t)
			authSrv.On("GetUserIDByToken", mock.Anything).
				Return(1, nil)
			m := middleware.MakeMiddleware(logger, authSrv)
			srv := server.NewServer(config.Config{}, h, m)

			reqBody := map[string]interface{}{
				"order": tt.args.order,
				"sum":   tt.args.sum,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer blatest")
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			r := srv.GetRouter()
			r.ServeHTTP(w, req)

			require.Equal(t, tt.want.status, w.Code)

			withdrawalService.AssertExpectations(t)
		})
	}
}
