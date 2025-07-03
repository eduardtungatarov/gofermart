package config

import (
	"flag"
	"os"
	"time"
)

const (
	DefaultServerHostPort = "localhost:8080"
	DefaultDatabaseDSN    = "host=localhost port=5432 user=myuser password=mypassword dbname=mydatabase sslmode=disable"
)

type Config struct {
	ServerHostPort string
	Database
}

type Database struct {
	DSN     string
	Timeout time.Duration
}

func Load() Config {
	flagServer := flag.String("a", DefaultServerHostPort, "отвечает за адрес запуска HTTP-сервера")
	databaseDSN := flag.String("d", DefaultDatabaseDSN, "строка с адресом подключения к БД")
	flag.Parse()

	aEnv, ok := os.LookupEnv("SERVER_ADDRESS")
	if ok {
		*flagServer = aEnv
	}

	dEnv, ok := os.LookupEnv("DATABASE_DSN")
	if ok {
		*databaseDSN = dEnv
	}

	return Config{
		ServerHostPort: *flagServer,
		Database: Database{
			DSN:     *databaseDSN,
			Timeout: time.Second * 1,
		},
	}
}
