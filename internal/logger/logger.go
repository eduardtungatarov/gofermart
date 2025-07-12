package logger

import "go.uber.org/zap"

func MakeLogger() (*zap.SugaredLogger, error) {
	log, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return log.Sugar(), nil
}
