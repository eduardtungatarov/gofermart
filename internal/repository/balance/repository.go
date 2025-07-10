package balance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/eduardtungatarov/gofermart/internal/repository"

	"github.com/eduardtungatarov/gofermart/internal/repository/balance/queries"
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

func (r *Repository) FindByUserID(ctx context.Context, userID int) (queries.Balance, error) {
	model, err := r.querier.FindByUserId(ctx, r.db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return queries.Balance{}, repository.ErrNoModel
		}
		return queries.Balance{}, fmt.Errorf("failed querier.FindByUserId: %w", err)
	}
	return model, err
}
