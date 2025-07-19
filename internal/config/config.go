package config

import (
	"flag"
	"os"
	"time"
)

type UserIDKey string

const (
	DefaultRunAddr                     = "localhost:8081"
	DefaultDatabaseURI                 = "host=localhost port=5432 user=myuser password=mypassword dbname=mydatabase sslmode=disable"
	DefaultAccrualSystemAdrr           = "http://localhost:8080"
	UserIDKeyName            UserIDKey = "userId"
)

type Config struct {
	RunAddr     string
	AccrualAddr string
	Database
	OrderPoll
	ShutdownTime time.Duration // Время, которое даем для корректного завершения сервиса.
}

type Database struct {
	DSN string
}

type OrderPoll struct {
	PollSleepTime time.Duration
	PollWorkerNum int
}

func Load() Config {
	runAddr := flag.String("a", DefaultRunAddr, "отвечает за адрес запуска HTTP-сервера")
	databaseURI := flag.String("d", DefaultDatabaseURI, "строка с адресом подключения к БД")
	accrualAddr := flag.String("r", DefaultAccrualSystemAdrr, "адрес системы расчёта начислений")
	flag.Parse()

	aEnv, ok := os.LookupEnv("RUN_ADDRESS")
	if ok {
		*runAddr = aEnv
	}

	dEnv, ok := os.LookupEnv("DATABASE_URI")
	if ok {
		*databaseURI = dEnv
	}

	rEnv, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS")
	if ok {
		*accrualAddr = rEnv
	}

	return Config{
		RunAddr:     *runAddr,
		AccrualAddr: *accrualAddr,
		Database: Database{
			DSN: *databaseURI,
		},
		OrderPoll: OrderPoll{
			PollSleepTime: time.Second * 5,
			PollWorkerNum: 10,
		},
		ShutdownTime: time.Second * 5,
	}
}
