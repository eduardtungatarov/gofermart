package order

import (
	"context"
	"fmt"

	"github.com/eduardtungatarov/gofermart/internal/utils"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal/queries"
)

//go:generate mockery --name=WithdrawalRepository
type WithdrawalRepository interface {
	FindByUserID(ctx context.Context, userID int) ([]queries.Withdrawal, error)
	SaveWithdrawal(ctx context.Context, withdrawal queries.Withdrawal) error
}

type Service struct {
	withdrawalRepo WithdrawalRepository
}

func New(withdrawalRepo WithdrawalRepository) *Service {
	return &Service{
		withdrawalRepo: withdrawalRepo,
	}
}

func (s *Service) GetUserWithdrawals(ctx context.Context) ([]queries.Withdrawal, error) {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed utils.GetUserID: %w", err)
	}

	withdrawals, err := s.withdrawalRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("withdrawalRepo.FindByUserId: %w", err)
	}

	return withdrawals, nil
}

func (s *Service) SaveWithdrawal(ctx context.Context, orderNumber string, sum int) error {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return fmt.Errorf("failed utils.GetUserID: %w", err)
	}

	return s.withdrawalRepo.SaveWithdrawal(ctx, queries.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         sum,
	})
}
