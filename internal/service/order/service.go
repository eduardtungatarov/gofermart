package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/eduardtungatarov/gofermart/internal/repository/order"
	"github.com/eduardtungatarov/gofermart/internal/utils"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"
)

const (
	STATUS_NEW = "NEW"
)

var (
	ErrOrderAlreadyUploadedByUser        = errors.New("order number was already uploaded by this user")
	ErrOrderAlreadyUploadedByAnotherUser = errors.New("order number was already uploaded by another user")
)

//go:generate mockery --name=OrderRepository
type OrderRepository interface {
	SaveOrder(ctx context.Context, order queries.Order) (queries.Order, error)
	FindOrderByOrderNumber(ctx context.Context, orderNumber string) (queries.Order, error)
}

type Service struct {
	orderRepo OrderRepository
}

func New(orderRepo OrderRepository) *Service {
	return &Service{
		orderRepo: orderRepo,
	}
}

func (s *Service) PostUserOrders(ctx context.Context, orderNumber string) error {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return err
	}

	_, err = s.orderRepo.SaveOrder(ctx, queries.Order{
		UserID:      userID,
		OrderNumber: orderNumber,
		Status:      STATUS_NEW,
		Accrual:     0,
	})

	if err != nil {
		if errors.Is(err, order.ErrOrderAlreadyExists) {
			orderModel, err := s.orderRepo.FindOrderByOrderNumber(ctx, orderNumber)
			if err != nil {
				return fmt.Errorf("failed FindOrderByOrderNumber: %w", err)
			}

			if orderModel.UserID == userID {
				return ErrOrderAlreadyUploadedByUser
			}

			return ErrOrderAlreadyUploadedByAnotherUser
		}

		return fmt.Errorf("failed to SaveOrder: %w", err)
	}

	return nil
}
