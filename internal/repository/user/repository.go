package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/eduardtungatarov/gofermart/internal/repository"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"
)

var ErrUserAlreadyExists = errors.New("user with this login already exists")

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

func (r *Repository) FindUserByLogin(ctx context.Context, login string) (queries.User, error) {
	model, err := r.querier.FindUserByLogin(ctx, r.db, login)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return queries.User{}, repository.ErrNoModel
		}
		return queries.User{}, fmt.Errorf("failed to find user by token: %w", err)
	}

	return model, nil
}

func (r *Repository) UpdateTokenByLogin(ctx context.Context, login, token string) (queries.User, error) {
	model, err := r.querier.UpdateTokenByUser(ctx, r.db, queries.UpdateTokenByUserParams{
		Login: login,
		Token: token,
	})

	if err != nil {
		return queries.User{}, fmt.Errorf("failed to update token by login: %w", err)
	}

	return model, nil
}
