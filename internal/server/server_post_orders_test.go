package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"github.com/eduardtungatarov/gofermart/internal/handlers/mocks"
	mocksMiddleware "github.com/eduardtungatarov/gofermart/internal/middleware/mocks"

	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	"github.com/eduardtungatarov/gofermart/internal/server"
	"github.com/eduardtungatarov/gofermart/internal/service/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestServer_PostUserOrders(t *testing.T) {
	type args struct {
		contentType string
		body        string
	}

	type mockBehavior func(os *mocks.OrderService, orderNumber string)

	tests := []struct {
		name           string
		args           args
		mockBehavior   mockBehavior
		expectedStatus int
		wantErr        bool
	}{
		{
			name: "successful_order_upload",
			args: args{
				contentType: "text/plain",
				body:        "49927398716", // Valid Luhn number
			},
			mockBehavior: func(os *mocks.OrderService, orderNumber string) {
				os.On("PostUserOrders", mock.Anything, orderNumber).Return(nil)
			},
			expectedStatus: http.StatusAccepted,
			wantErr:        false,
		},
		{
			name: "order_already_uploaded_by_user",
			args: args{
				contentType: "text/plain",
				body:        "49927398716",
			},
			mockBehavior: func(os *mocks.OrderService, orderNumber string) {
				os.On("PostUserOrders", mock.Anything, orderNumber).
					Return(order.ErrOrderAlreadyUploadedByUser)
			},
			expectedStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "order_already_uploaded_by_another_user",
			args: args{
				contentType: "text/plain",
				body:        "49927398716",
			},
			mockBehavior: func(os *mocks.OrderService, orderNumber string) {
				os.On("PostUserOrders", mock.Anything, orderNumber).
					Return(order.ErrOrderAlreadyUploadedByAnotherUser)
			},
			expectedStatus: http.StatusConflict,
			wantErr:        false,
		},
		{
			name: "invalid_content_type",
			args: args{
				contentType: "application/json",
				body:        "49927398716",
			},
			mockBehavior:   func(os *mocks.OrderService, orderNumber string) {},
			expectedStatus: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name: "empty_body",
			args: args{
				contentType: "text/plain",
				body:        "",
			},
			mockBehavior:   func(os *mocks.OrderService, orderNumber string) {},
			expectedStatus: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name: "invalid_Luhn_number",
			args: args{
				contentType: "text/plain",
				body:        "49927398717", // Invalid Luhn number
			},
			mockBehavior:   func(os *mocks.OrderService, orderNumber string) {},
			expectedStatus: http.StatusUnprocessableEntity,
			wantErr:        true,
		},
		{
			name: "non-numeric_order_number",
			args: args{
				contentType: "text/plain",
				body:        "123abc456",
			},
			mockBehavior:   func(os *mocks.OrderService, orderNumber string) {},
			expectedStatus: http.StatusUnprocessableEntity,
			wantErr:        true,
		},
		{
			name: "internal_server_error",
			args: args{
				contentType: "text/plain",
				body:        "49927398716",
			},
			mockBehavior: func(os *mocks.OrderService, orderNumber string) {
				os.On("PostUserOrders", mock.Anything, orderNumber).
					Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			orderService := mocks.NewOrderService(t)
			tt.mockBehavior(orderService, tt.args.body)

			logger := zap.NewNop().Sugar()
			authSrv := mocksMiddleware.NewAuthService(t)
			authSrv.On("GetUserIDByToken", mock.Anything).
				Return(1, nil)

			m := middleware.MakeMiddleware(logger, authSrv)
			h := handlers.MakeHandler(logger, nil, orderService, nil, nil)
			srv := server.NewServer(config.Config{}, h, m)

			req := httptest.NewRequest("POST", "/api/user/orders", bytes.NewBufferString(tt.args.body))
			req.Header.Set("Content-Type", tt.args.contentType)
			req.Header.Set("Authorization", "Bearer blatest")

			rec := httptest.NewRecorder()

			// Act
			srv.GetRouter().ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if !tt.wantErr {
				orderService.AssertExpectations(t)
			}
		})
	}
}
