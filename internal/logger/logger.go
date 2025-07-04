package logger

import "go.uber.org/zap"

func MakeLogger() (*zap.SugaredLogger, error) {
	log, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return log.Sugar(), nil
}

func MakeNop() (*zap.SugaredLogger, error) {
	log := zap.NewNop()
	return log.Sugar(), nil
}
