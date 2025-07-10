package config

import (
	"flag"
	"os"
	"time"
)

type UserIDKey string

const (
	DefaultRunADDR                     = "localhost:8081"
	DefaultDatabaseURI                 = "host=localhost port=5432 user=myuser password=mypassword dbname=mydatabase sslmode=disable"
	DefaultAccrualSystemADRR           = ""
	UserIDKeyName            UserIDKey = "userId"
)

type Config struct {
	RunADDR     string
	AccrualADDR string
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
	runADDR := flag.String("a", DefaultRunADDR, "отвечает за адрес запуска HTTP-сервера")
	databaseURI := flag.String("d", DefaultDatabaseURI, "строка с адресом подключения к БД")
	accrualADDR := flag.String("r", DefaultAccrualSystemADRR, "адрес системы расчёта начислений")
	flag.Parse()

	aEnv, ok := os.LookupEnv("RUN_ADDRESS")
	if ok {
		*runADDR = aEnv
	}

	dEnv, ok := os.LookupEnv("DATABASE_URI")
	if ok {
		*databaseURI = dEnv
	}

	rEnv, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS")
	if ok {
		*accrualADDR = rEnv
	}

	return Config{
		RunADDR:     *runADDR,
		AccrualADDR: *accrualADDR,
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
