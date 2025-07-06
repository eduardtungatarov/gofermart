package auth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"
	"github.com/google/uuid"
)

var ErrLoginPwd = errors.New("invalid username/password pair")

//go:generate mockery --name=UserRepository
type UserRepository interface {
	SaveUser(ctx context.Context, user queries.User) (queries.User, error)
	FindUserByLogin(ctx context.Context, login string) (queries.User, error)
	UpdateTokenByLogin(ctx context.Context, login, token string) (queries.User, error)
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
	hashedPassword, err := s.getHashedPwd(pwd)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	token := s.generateTokenAndGet()

	insertedUser, err := s.userRepo.SaveUser(ctx, queries.User{
		Login:    login,
		Password: hashedPassword,
		Token:    token,
	})
	if err != nil {
		return "", err
	}

	return insertedUser.Token, nil
}

func (s *Service) Login(ctx context.Context, login, pwd string) (string, error) {
	user, err := s.userRepo.FindUserByLogin(ctx, login)
	if err != nil {
		return "", ErrLoginPwd
	}

	if !s.checkPasswordHash(pwd, user.Password) {
		return "", ErrLoginPwd
	}

	token := s.generateTokenAndGet()

	_, err = s.userRepo.UpdateTokenByLogin(ctx, user.Login, token)
	if err != nil {
		return "", err
	}

	return user.Token, nil
}

func (s *Service) getHashedPwd(pwd string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), 14)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (s *Service) checkPasswordHash(pwd, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	return err == nil
}

func (s *Service) generateTokenAndGet() string {
	return uuid.NewString()
}
