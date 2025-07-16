package orderpoll

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"

	"github.com/eduardtungatarov/gofermart/internal/accrual"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/eduardtungatarov/gofermart/internal/orderpoll/mocks"

	"github.com/eduardtungatarov/gofermart/internal/config"
	"go.uber.org/zap"
)

func TestOrderPoll_RunWorker(t *testing.T) {
	tests := []struct {
		name            string
		clientMockSetup func(m *mocks.AccrualClient)
		orderMockSetup  func(m *mocks.OrderService)
		cancelCtx       bool
		expectError     bool
		inputOrders     []OrderChValue
	}{
		{
			name: "success_order_found",
			clientMockSetup: func(m *mocks.AccrualClient) {
				m.EXPECT().GetOrder("346436439").
					Return(&accrual.Order{
						Order:   "346436439",
						Status:  "PROCESSED",
						Accrual: 100.55,
					}, nil)
			},
			orderMockSetup: func(m *mocks.OrderService) {
				m.On("UpdateOrder", mock.Anything, 1, "346436439", "PROCESSED", 10055).
					Return(nil)
			},
			cancelCtx:   false,
			expectError: false,
			inputOrders: []OrderChValue{
				{
					OrderNumber: "346436439",
					UserID:      1,
				},
			},
		},
		{
			name: "accrual_net_error",
			clientMockSetup: func(m *mocks.AccrualClient) {
				m.EXPECT().GetOrder("346436439").
					Return(nil, errors.New("net error")) // сетевая ошибка.
			},
			orderMockSetup: func(m *mocks.OrderService) {
				//
			},
			cancelCtx:   false,
			expectError: true,
			inputOrders: []OrderChValue{
				{
					OrderNumber: "346436439",
					UserID:      1,
				},
			},
		},
		{
			name: "order_update_error",
			clientMockSetup: func(m *mocks.AccrualClient) {
				m.EXPECT().GetOrder("346436439").
					Return(&accrual.Order{
						Order:   "346436439",
						Status:  "PROCESSED",
						Accrual: 100.55,
					}, nil)
			},
			orderMockSetup: func(m *mocks.OrderService) {
				m.EXPECT().UpdateOrder(mock.Anything, 1, "346436439", "PROCESSED", 10055).
					Return(errors.New("net error")) // сетевая ошибка.
			},
			cancelCtx:   false,
			expectError: true,
			inputOrders: []OrderChValue{
				{
					OrderNumber: "346436439",
					UserID:      1,
				},
			},
		},
		{
			name: "accrual_no_content_status",
			clientMockSetup: func(m *mocks.AccrualClient) {
				m.EXPECT().GetOrder("346436439").
					Return(nil, &accrual.NonOkError{Code: http.StatusNoContent}) // 204 status answer
			},
			orderMockSetup: func(m *mocks.OrderService) {
				m.EXPECT().UpdateOrder(mock.Anything, 1, "346436439", "INVALID", 0). // then status INVALID set
													Return(nil)
			},
			cancelCtx:   false,
			expectError: false,
			inputOrders: []OrderChValue{
				{
					OrderNumber: "346436439",
					UserID:      1,
				},
			},
		},
		{
			name: "channel_is_closed",
			clientMockSetup: func(m *mocks.AccrualClient) {
				//
			},
			orderMockSetup: func(m *mocks.OrderService) {
				//
			},
			cancelCtx:   false,
			expectError: false,
			inputOrders: nil, // empty, channel auto close
		},
		{
			name: "context_is_canceled",
			clientMockSetup: func(m *mocks.AccrualClient) {
				//
			},
			orderMockSetup: func(m *mocks.OrderService) {
				//
			},
			cancelCtx:   true,
			expectError: false,
			inputOrders: nil, // empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготовка.
			orderSrv := mocks.NewOrderService(t)
			client := mocks.NewAccrualClient(t)
			tt.clientMockSetup(client)
			tt.orderMockSetup(orderSrv)
			o := &OrderPoll{
				orderSrv: orderSrv,
				client:   client,
				// nop
				log:       zap.NewNop().Sugar(),
				cfg:       config.Config{},
				sleepTime: time.Second * 0,
				workerNum: 0,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ch := make(chan OrderChValue, len(tt.inputOrders))
			for _, order := range tt.inputOrders {
				ch <- order
			}
			close(ch)

			// Проверяем.
			if tt.cancelCtx {
				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()
			}
			err := o.RunWorker(ctx, ch)
			if tt.expectError {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "nil error expected")
			}
		})
	}
}

func TestOrderPoll_RunReader(t *testing.T) {
	tests := []struct {
		name           string
		orderMockSetup func(m *mocks.OrderService)
		expectError    bool
		expectOutput   []OrderChValue
	}{
		{
			name: "success_order_found",
			orderMockSetup: func(m *mocks.OrderService) {
				m.On("FindByInProgressStatuses", mock.Anything).
					Return([]queries.Order{
						{
							OrderNumber: "123",
							UserID:      1,
						},
						{
							OrderNumber: "456",
							UserID:      2,
						},
					}, nil)
			},
			expectError: false,
			expectOutput: []OrderChValue{
				{
					OrderNumber: "123",
					UserID:      1,
				},
				{
					OrderNumber: "456",
					UserID:      2,
				},
			},
		},
		{
			name: "err_order_found",
			orderMockSetup: func(m *mocks.OrderService) {
				m.On("FindByInProgressStatuses", mock.Anything).
					Return([]queries.Order{}, errors.New("db error"))
			},
			expectError:  true,
			expectOutput: []OrderChValue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготовка.
			orderSrv := mocks.NewOrderService(t)
			tt.orderMockSetup(orderSrv)
			o := &OrderPoll{
				orderSrv: orderSrv,
				client:   nil,
				// nop
				log:       zap.NewNop().Sugar(),
				cfg:       config.Config{},
				sleepTime: time.Millisecond * 1,
				workerNum: 0,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Проверяем.
			cancel() // чтобы выйти из бесконеч цикла.
			output := make(chan OrderChValue, len(tt.expectOutput))
			err := o.RunReader(ctx, output)
			if tt.expectError {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "nil error expected")
			}

			if len(tt.expectOutput) > 0 {
				for order := range output {
					assert.Contains(t, tt.expectOutput, order, "order %v not found in output channel = %v", order, tt.expectOutput)
				}
			}

			orderSrv.AssertExpectations(t)
		})
	}
}
