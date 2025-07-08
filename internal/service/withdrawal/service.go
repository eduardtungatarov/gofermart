package order

import (
	"context"
	"fmt"

	"github.com/eduardtungatarov/gofermart/internal/utils"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal/queries"
)

//go:generate mockery --name=WithdrawalRepository
type WithdrawalRepository interface {
	FindByUserId(ctx context.Context, userID int) ([]queries.Withdrawal, error)
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

	withdrawals, err := s.withdrawalRepo.FindByUserId(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("withdrawalRepo.FindByUserId: %w", err)
	}

	return withdrawals, nil
}
