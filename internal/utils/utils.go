package utils

import (
	"context"
	"errors"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

func GetUserID(ctx context.Context) (int, error) {
	if userID, ok := ctx.Value(config.UserIDKeyName).(int); ok {
		return userID, nil
	}
	return 0, errors.New("userID not found or not a string")
}
