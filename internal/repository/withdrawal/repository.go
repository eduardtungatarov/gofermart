package withdrawal

import (
	"context"
	"fmt"

	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal/queries"
)

type Repository struct {
	db      queries.DBTX
	querier queries.Querier
}

func New(db queries.DBTX) *Repository {
	return &Repository{
		db:      db,
		querier: queries.New(),
	}
}

func (r *Repository) FindByUserId(ctx context.Context, userID int) ([]queries.Withdrawal, error) {
	models, err := r.querier.FindByUserId(ctx, r.db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed querier.FindByUserId: %w", err)
	}
	return models, nil
}
