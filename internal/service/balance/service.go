package balance

import (
	"context"
	"errors"
	"fmt"

	"github.com/eduardtungatarov/gofermart/internal/repository"

	"github.com/eduardtungatarov/gofermart/internal/utils"

	"github.com/eduardtungatarov/gofermart/internal/repository/balance/queries"
)

//go:generate mockery --name=BalanceRepository
type BalanceRepository interface {
	FindByUserID(ctx context.Context, userID int) (queries.Balance, error)
}

type Service struct {
	balanceRepo BalanceRepository
}

func New(balanceRepo BalanceRepository) *Service {
	return &Service{
		balanceRepo: balanceRepo,
	}
}

func (s *Service) GetUserBalance(ctx context.Context) (queries.Balance, error) {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return queries.Balance{}, fmt.Errorf("failed GetUserID from ctx: %w", err)
	}

	balance, err := s.balanceRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNoModel) {
			return queries.Balance{}, nil
		}
		return queries.Balance{}, fmt.Errorf("failed balanceRepo.FindByUserId: %w", err)
	}

	return balance, nil
}
