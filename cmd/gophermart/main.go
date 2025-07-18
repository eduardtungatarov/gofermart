package main

import (
	"context"
	"database/sql"
	stlog "log"
	"os/signal"
	"sync"
	"syscall"

	"github.com/eduardtungatarov/gofermart/internal/accrual"

	"github.com/eduardtungatarov/gofermart/internal/orderpoll"

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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Создаем логер.
	log, err := logger.MakeLogger()
	if err != nil {
		stlog.Fatalf("Failed to make Logger: %v", err)
	}

	// Создаем конфиг.
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

	// Собираем зависимости.
	userRepo := userRepository.New(db)
	orderRepo := orderRepository.New(db)
	balanceRepo := balanceRepository.New(db)
	withdrawalRepo := withdrawalRepository.New(db)
	authSrv := authService.New(userRepo)
	orderSrv := orderService.New(orderRepo)
	balanceSrv := balanceService.New(balanceRepo)
	withdrawalSrv := withdrawalService.New(withdrawalRepo)

	// Настраиваем опрашиватель.
	client := accrual.NewClient(cfg)
	op := orderpoll.New(log, cfg, orderSrv, client)

	// Настраиваем сервер.
	h := handlers.MakeHandler(log, authSrv, orderSrv, balanceSrv, withdrawalSrv)
	m := middleware.MakeMiddleware(log, authSrv)
	s := server.NewServer(cfg, h, m)

	wg := sync.WaitGroup{}
	wg.Add(2)

	// Запускаем опрашиватель заказов.
	go func() {
		defer wg.Done()
		op.Run(ctx)
	}()
	// Запускаем сервер.
	go func() {
		defer wg.Done()
		err = s.Run(ctx)
		if err != nil {
			log.Errorf("httpserver.Run() failed: %w", err)
		}
	}()
	log.Info("Service started")

	<-ctx.Done()
	log.Info("Service stop...")
	wg.Wait()
	log.Info("Service stopped")
}
