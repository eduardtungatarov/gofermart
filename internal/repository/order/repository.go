package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	balanceQueries "github.com/eduardtungatarov/gofermart/internal/repository/balance/queries"

	"github.com/eduardtungatarov/gofermart/internal/repository"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eduardtungatarov/gofermart/internal/repository/order/queries"
)

var ErrOrderAlreadyExists = errors.New("order with this number already exists")

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

func (r *Repository) SaveOrder(ctx context.Context, order queries.Order) (queries.Order, error) {
	model, err := r.querier.SaveOrder(ctx, r.db, queries.SaveOrderParams{
		UserID:      order.UserID,
		OrderNumber: order.OrderNumber,
		Status:      order.Status,
		Accrual:     order.Accrual,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return queries.Order{}, ErrOrderAlreadyExists
		}
		return queries.Order{}, fmt.Errorf("failed to save order: %w", err)
	}

	return model, err
}

func (r *Repository) FindOrderByOrderNumber(ctx context.Context, orderNumber string) (queries.Order, error) {
	model, err := r.querier.FindOrderByOrderNumber(ctx, r.db, orderNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return queries.Order{}, repository.ErrNoModel
		}
		return queries.Order{}, fmt.Errorf("failed to find order by orderNumber: %w", err)
	}

	return model, nil
}

func (r *Repository) FindByUserID(ctx context.Context, userID int) ([]queries.Order, error) {
	models, err := r.querier.FindByUserId(ctx, r.db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find order by userID: %w", err)
	}
	return models, nil
}

func (r *Repository) FindByInProgressStatuses(ctx context.Context) ([]queries.Order, error) {
	models, err := r.querier.FindByInProgressStatuses(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("querier.FindByInProgressStatuses: %w", err)
	}
	return models, nil
}

func (r *Repository) UpdateOrder(ctx context.Context, userID int, orderNumber, status string, accrual int) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("db does not support transactions")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTx: %w", err)
	}
	defer tx.Rollback()

	_, err = r.querier.UpdateOrder(ctx, tx, queries.UpdateOrderParams{
		OrderNumber: orderNumber,
		Status:      status,
		Accrual:     accrual,
	})
	if err != nil {
		return fmt.Errorf("querier.UpdateOrder: %w", err)
	}

	_, err = r.balanceQuerier.AddBalance(ctx, tx, balanceQueries.AddBalanceParams{
		Sum:    accrual,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("querier.UpdateOrder: %w", err)
	}

	return tx.Commit()
}
