package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardtungatarov/gofermart/internal/repository/user/queries"
)

const (
	secretKey     = "supersecretkey"
	tokenLifeTime = 24 * time.Hour
)

var ErrLoginPwd = errors.New("invalid username/password pair")

//go:generate mockery --name=UserRepository
type UserRepository interface {
	SaveUser(ctx context.Context, user queries.User) (queries.User, error)
	FindUserByLogin(ctx context.Context, login string) (queries.User, error)
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int
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

	user, err := s.userRepo.SaveUser(ctx, queries.User{
		Login:    login,
		Password: hashedPassword,
	})
	if err != nil {
		return "", err
	}

	token, err := s.buildJWTString(int(user.ID))
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) Login(ctx context.Context, login, pwd string) (string, error) {
	user, err := s.userRepo.FindUserByLogin(ctx, login)
	if err != nil {
		return "", ErrLoginPwd
	}

	if !s.checkPasswordHash(pwd, user.Password) {
		return "", ErrLoginPwd
	}

	token, err := s.buildJWTString(int(user.ID))
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) GetUserIDByToken(tokenStr string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		})
	if err != nil {
		return 0, err
	}

	if !token.Valid {
		return 0, errors.New("auth token is not valid")
	}

	return claims.UserID, nil
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

func (s *Service) buildJWTString(userID int) (string, error) {
	expirationTime := time.Now().Add(tokenLifeTime)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
