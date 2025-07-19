package orderpoll

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/eduardtungatarov/gofermart/internal/service/order"

	"github.com/eduardtungatarov/gofermart/internal/accrual"

	"github.com/eduardtungatarov/gofermart/internal/config"

	"go.uber.org/zap"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"
)

//go:generate mockery --with-expecter --name=OrderService
type OrderService interface {
	FindByInProgressStatuses(ctx context.Context) ([]queries.Order, error)
	UpdateOrder(ctx context.Context, userID int, orderNumber, status string, accrual int) error
}

//go:generate mockery --with-expecter --name=AccrualClient
type AccrualClient interface {
	GetOrder(orderNumber string) (*accrual.Order, error)
}

type OrderChValue struct {
	OrderNumber string
	UserID      int
}

type OrderPoll struct {
	log                 *zap.SugaredLogger
	cfg                 config.Config
	orderSrv            OrderService
	sleepTime           time.Duration
	workerNum           int
	client              AccrualClient
	reqsBlockedUntil    time.Time
	reqsBlockedUntilMtx sync.RWMutex
}

func New(log *zap.SugaredLogger, cfg config.Config, orderSrv OrderService, client AccrualClient) *OrderPoll {
	return &OrderPoll{
		log:                 log,
		cfg:                 cfg,
		orderSrv:            orderSrv,
		sleepTime:           cfg.OrderPoll.PollSleepTime,
		workerNum:           cfg.OrderPoll.PollWorkerNum,
		client:              client,
		reqsBlockedUntil:    time.Now(),
		reqsBlockedUntilMtx: sync.RWMutex{},
	}
}

func (o *OrderPoll) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	orderCh := make(chan OrderChValue, o.workerNum)

	wg := sync.WaitGroup{}
	wg.Add(o.workerNum + 1)

	go func() {
		defer wg.Done()
		o.RunReader(ctx, orderCh)
	}()

	for i := 0; i < o.workerNum; i++ {
		go func() {
			defer wg.Done()
			o.RunWorker(ctx, orderCh)
		}()
	}

	<-ctx.Done()
	wg.Wait()
}

func (o *OrderPoll) RunReader(ctx context.Context, ch chan<- OrderChValue) {
	defer close(ch)
	for {
		orders, err := o.orderSrv.FindByInProgressStatuses(ctx)
		if err != nil {
			o.log.Errorf("RunReader o.orderSrv.FindByInProgressStatuses: %w", err)
			continue
		}

		for _, v := range orders {
			select {
			case ch <- OrderChValue{
				OrderNumber: v.OrderNumber,
				UserID:      v.UserID,
			}:
			case <-ctx.Done():
				return
			}
		}

		select {
		case <-time.After(o.sleepTime):
		case <-ctx.Done():
			return
		}
	}
}

func (o *OrderPoll) RunWorker(ctx context.Context, ch <-chan OrderChValue) {
	for {
		o.reqsBlockedUntilMtx.RLock()
		waitUntil := o.reqsBlockedUntil
		o.reqsBlockedUntilMtx.RUnlock()
		if now := time.Now(); now.Before(waitUntil) {
			waitTime := waitUntil.Sub(now)
			select {
			case <-time.After(waitTime):
				continue
			case <-ctx.Done():
				return
			}
		}

		select {
		case orderChV, ok := <-ch:
			if !ok {
				return
			}

			resp, err := o.client.GetOrder(orderChV.OrderNumber)
			if err != nil {
				var nonOkErr *accrual.NonOkError
				if ok := errors.As(err, &nonOkErr); ok {
					if nonOkErr.Code == http.StatusNoContent {
						err = o.orderSrv.UpdateOrder(ctx, orderChV.UserID, orderChV.OrderNumber, order.StatusInvalid, 0)
						if err != nil {
							o.log.Errorf("RunWorker o.orderSrv.UpdateOrder: %w", err)
							continue
						}
					}
					if nonOkErr.Code == http.StatusTooManyRequests {
						o.reqsBlockedUntilMtx.Lock()
						o.reqsBlockedUntil = time.Now().Add(time.Second * time.Duration(nonOkErr.RetryAfter))
						o.reqsBlockedUntilMtx.Unlock()
					}
				}
				continue
			}

			err = o.orderSrv.UpdateOrder(ctx, orderChV.UserID, orderChV.OrderNumber, resp.Status, int(resp.Accrual*100))
			if err != nil {
				o.log.Errorf("RunWorker o.orderSrv.UpdateOrder: %w", err)
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
