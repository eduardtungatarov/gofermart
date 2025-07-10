package orderpoll

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/eduardtungatarov/gofermart/internal/service/order"

	"github.com/eduardtungatarov/gofermart/internal/accrual"

	"github.com/eduardtungatarov/gofermart/internal/config"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"
)

//go:generate mockery --name=OrderService
type OrderService interface {
	FindByInProgressStatuses(ctx context.Context) ([]queries.Order, error)
	UpdateOrder(ctx context.Context, userID int, orderNumber, status string, accrual int) error
}

type AccrualClient interface {
	GetOrder(orderNumber string) (*accrual.Order, error)
}

type OrderChValue struct {
	OrderNumber string
	UserID      int
}

type OrderPoll struct {
	log       *zap.SugaredLogger
	cfg       config.Config
	orderSrv  OrderService
	sleepTime time.Duration
	workerNum int
	client    AccrualClient
}

func New(log *zap.SugaredLogger, cfg config.Config, orderSrv OrderService, client AccrualClient) *OrderPoll {
	return &OrderPoll{
		log:       log,
		cfg:       cfg,
		orderSrv:  orderSrv,
		sleepTime: cfg.OrderPoll.PollSleepTime,
		workerNum: cfg.OrderPoll.PollWorkerNum,
		client:    client,
	}
}

func (o *OrderPoll) Run(ctx context.Context) error {
	pollCtx, cancel := context.WithCancel(ctx)
	errChan := make(chan error, o.workerNum+1)
	orderCh := make(chan OrderChValue, o.workerNum)

	go func(ch chan<- OrderChValue) {
		defer close(ch)
		for {
			orders, err := o.orderSrv.FindByInProgressStatuses(pollCtx)
			if err != nil {
				errChan <- fmt.Errorf("orderSrv.FindByInProgressStatuses: %w", err)
				return
			}

			for _, v := range orders {
				select {
				case ch <- OrderChValue{
					OrderNumber: v.OrderNumber,
					UserID:      v.UserID,
				}:
				case <-pollCtx.Done():
					return
				}
			}

			select {
			case <-time.After(o.sleepTime):
			case <-pollCtx.Done():
				return
			}
		}
	}(orderCh)

	for i := 0; i < o.workerNum; i++ {
		go func(ch <-chan OrderChValue) {
			for {
				select {
				case orderChV, ok := <-orderCh:
					if !ok {
						return
					}
					//

					resp, err := o.client.GetOrder(orderChV.OrderNumber)
					if err != nil {
						var nonOkErr *accrual.NonOkError
						if ok := errors.As(err, &nonOkErr); ok {
							if nonOkErr.Code == http.StatusNoContent {
								err = o.orderSrv.UpdateOrder(pollCtx, orderChV.UserID, orderChV.OrderNumber, order.StatusInvalid, 0)
								if err != nil {
									errChan <- fmt.Errorf("orderSrv.UpdateOrder: %w", err)
									return
								}
							}
						}
						o.log.Error("o.client.GetOrder net err", err)
						return
					}

					err = o.orderSrv.UpdateOrder(pollCtx, orderChV.UserID, orderChV.OrderNumber, resp.Status, int(resp.Accrual*100))
					if err != nil {
						errChan <- fmt.Errorf("orderSrv.UpdateOrder: %w", err)
						return
					}

					//
				case <-pollCtx.Done():
					return
				}
			}
		}(orderCh)
	}

	select {
	case err := <-errChan:
		cancel()
		return err
	case <-pollCtx.Done():
		cancel()
		return nil
	}
}
