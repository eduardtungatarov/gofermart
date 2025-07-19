package withdrawal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	balanceQueries "github.com/eduardtungatarov/gofermart/internal/repository/balance/queries"
	"github.com/eduardtungatarov/gofermart/internal/repository/withdrawal/queries"
)

var ErrNoMoney = errors.New("there are insufficient funds in the account")

type Repository struct {
	db             queries.DBTX
	querier        queries.Querier
	balanceQuerier balanceQueries.Querier
}

func New(db queries.DBTX) *Repository {
	return &Repository{
		db:             db,
		querier:        queries.New(),
		balanceQuerier: balanceQueries.New(),
	}
}

func (r *Repository) FindByUserID(ctx context.Context, userID int) ([]queries.Withdrawal, error) {
	models, err := r.querier.FindByUserId(ctx, r.db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed querier.FindByUserId: %w", err)
	}
	return models, nil
}

func (r *Repository) SaveWithdrawal(ctx context.Context, withdrawal queries.Withdrawal) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("db does not support transactions")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTx: %w", err)
	}
	defer tx.Rollback()

	_, err = r.querier.SaveWithdrawal(ctx, tx, queries.SaveWithdrawalParams{
		UserID:      withdrawal.UserID,
		OrderNumber: withdrawal.OrderNumber,
		Sum:         withdrawal.Sum,
	})
	if err != nil {
		return fmt.Errorf("querier.SaveWithdrawal: %w", err)
	}

	_, err = r.balanceQuerier.DeductFromBalance(ctx, tx, balanceQueries.DeductFromBalanceParams{
		Sum:    withdrawal.Sum,
		UserID: withdrawal.UserID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23514" { // Код ошибки CHECK-ограничения
				return ErrNoMoney
			}
		}
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoMoney
		}
		return fmt.Errorf("balanceQuerier.DeductFromBalance: %w", err)
	}

	return tx.Commit()
}
