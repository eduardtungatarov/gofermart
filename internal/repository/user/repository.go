package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"
)

var (
	ErrUserAlreadyExists = errors.New("user with this login already exists")
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

func (r *Repository) SaveUser(ctx context.Context, user queries.User) (queries.User, error) {
	model, err := r.querier.SaveUser(ctx, r.db, queries.SaveUserParams{
		Login:    user.Login,
		Password: user.Password,
		Token:    user.Token,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return queries.User{}, ErrUserAlreadyExists
		}
		return queries.User{}, fmt.Errorf("failed to save user: %w", err)
	}

	return model, err
}
