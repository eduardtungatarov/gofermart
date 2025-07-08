package main

import (
	"database/sql"
	stlog "log"

	"github.com/eduardtungatarov/gofermart/internal/handlers"
	"github.com/eduardtungatarov/gofermart/internal/logger"
	"github.com/eduardtungatarov/gofermart/internal/middleware"
	balanceRepository "github.com/eduardtungatarov/gofermart/internal/repository/balance"
	orderRepository "github.com/eduardtungatarov/gofermart/internal/repository/order"
	userRepository "github.com/eduardtungatarov/gofermart/internal/repository/user"
	withdrawalRepository "github.com/eduardtungatarov/gofermart/internal/repository/withdrawal"
	"github.com/eduardtungatarov/gofermart/internal/server"
	authService "github.com/eduardtungatarov/gofermart/internal/service/auth"
	balanceService "github.com/eduardtungatarov/gofermart/internal/service/balance"
	orderService "github.com/eduardtungatarov/gofermart/internal/service/order"
	withdrawalService "github.com/eduardtungatarov/gofermart/internal/service/withdrawal"

	"github.com/eduardtungatarov/gofermart/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	log, err := logger.MakeLogger()
	if err != nil {
		stlog.Fatalf("Failed to make Logger: %v", err)
	}

	cfg := config.Load()

	// Подключаемся к базе данных.
	db, err := sql.Open("pgx", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Применяем миграции.
	err = goose.Up(db, "migrations")
	if err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	// Собираем дерево зависимостей.
	userRepo := userRepository.New(db)
	orderRepo := orderRepository.New(db)
	balanceRepo := balanceRepository.New(db)
	withdrawalRepo := withdrawalRepository.New(db)
	authSrv := authService.New(userRepo)
	orderSrv := orderService.New(orderRepo)
	balanceSrv := balanceService.New(balanceRepo)
	withdrawalSrv := withdrawalService.New(withdrawalRepo)

	// Запускаем сервер, указываем хендлеры и миддлваре.
	h := handlers.MakeHandler(log, authSrv, orderSrv, balanceSrv, withdrawalSrv)
	m := middleware.MakeMiddleware(log, authSrv)
	s := server.NewServer(cfg, h, m)
	err = s.Run()
	if err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
