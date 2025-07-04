package auth

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"
	"github.com/google/uuid"
)

const tokenLength = 150

type UserRepository interface {
	SaveUser(ctx context.Context, user queries.User) (queries.User, error)
}

type Service struct {
	userRepo UserRepository
}

func New(userRepo UserRepository) *Service {
	return &Service{
		userRepo: userRepo,
	}
}

func (s *Service) Register(ctx context.Context, login, pwd string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	token := uuid.NewString()

	_, err = s.userRepo.SaveUser(ctx, queries.User{
		Login:    login,
		Password: pwd,
		Token:    string(hashedPassword),
	})
	if err != nil {
		return "", err
	}

	return token, nil
}
