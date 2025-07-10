package orderpoll

import (
	"context"
	"fmt"
	"time"

	"github.com/eduardtungatarov/gofermart/internal/accrual"

	"github.com/eduardtungatarov/gofermart/internal/config"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"
)

//go:generate mockery --name=OrderService
type OrderService interface {
	FindByInProgressStatuses(ctx context.Context) ([]queries.Order, error)
	UpdateOrder(ctx context.Context, orderNumber, status string, accrual int) error
}

type AccrualClient interface {
	GetOrder(orderNumber string) (*accrual.Order, error)
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
	orderNumberCh := make(chan string, o.workerNum)

	go func(ch chan<- string) {
		defer close(ch)
		for {
			orders, err := o.orderSrv.FindByInProgressStatuses(pollCtx)
			if err != nil {
				errChan <- fmt.Errorf("orderSrv.FindByInProgressStatuses: %w", err)
				return
			}

			for _, v := range orders {
				select {
				case ch <- v.OrderNumber:
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
	}(orderNumberCh)

	for i := 0; i < o.workerNum; i++ {
		go func(ch <-chan string) {
			for {
				select {
				case orderNum, ok := <-orderNumberCh:
					if !ok {
						return
					}

					resp, err := o.client.GetOrder(orderNum)
					if err != nil {
						o.log.Error("o.client.GetOrder err", err)
						return
					}

					err = o.orderSrv.UpdateOrder(pollCtx, orderNum, resp.Status, resp.Accrual)
					if err != nil {
						errChan <- fmt.Errorf("orderSrv.UpdateOrder: %w", err)
						return
					}
				case <-pollCtx.Done():
					return
				}
			}
		}(orderNumberCh)
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
