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
	ctx, cancel := context.WithCancel(ctx)

	errChan := make(chan error, o.workerNum+1)
	orderCh := make(chan OrderChValue, o.workerNum)

	go func() {
		err := o.RunReader(ctx, orderCh)
		if err != nil {
			errChan <- fmt.Errorf("orderSrv.UpdateOrder: %w", err)
		}
	}()

	for i := 0; i < o.workerNum; i++ {
		go func() {
			err := o.RunWorker(ctx, orderCh)
			if err != nil {
				errChan <- fmt.Errorf("o.RunWorker: %w", err)
			}
		}()
	}

	select {
	case err := <-errChan:
		cancel()
		return err
	case <-ctx.Done():
		cancel()
		return nil
	}
}

func (o *OrderPoll) RunReader(ctx context.Context, ch chan<- OrderChValue) error {
	defer close(ch)
	for {
		orders, err := o.orderSrv.FindByInProgressStatuses(ctx)
		if err != nil {
			return err
		}

		for _, v := range orders {
			select {
			case ch <- OrderChValue{
				OrderNumber: v.OrderNumber,
				UserID:      v.UserID,
			}:
			case <-ctx.Done():
				return nil
			}
		}

		select {
		case <-time.After(o.sleepTime):
		case <-ctx.Done():
			return nil
		}
	}
}

func (o *OrderPoll) RunWorker(ctx context.Context, ch <-chan OrderChValue) error {
	for {
		select {
		case orderChV, ok := <-ch:
			if !ok {
				return nil
			}

			resp, err := o.client.GetOrder(orderChV.OrderNumber)
			if err != nil {
				var nonOkErr *accrual.NonOkError
				if ok := errors.As(err, &nonOkErr); ok {
					if nonOkErr.Code == http.StatusNoContent {
						err = o.orderSrv.UpdateOrder(ctx, orderChV.UserID, orderChV.OrderNumber, order.StatusInvalid, 0)
						if err != nil {
							return err
						}
						continue
					}
				}
				return err
			}

			err = o.orderSrv.UpdateOrder(ctx, orderChV.UserID, orderChV.OrderNumber, resp.Status, int(resp.Accrual*100))
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
