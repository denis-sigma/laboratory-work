package main

import (
	"context"
	"github.com/denis-sigma/laboratory-work/internal/config"
	"github.com/denis-sigma/laboratory-work/internal/database"
	api "github.com/denis-sigma/laboratory-work/internal/generated"
	"github.com/denis-sigma/laboratory-work/internal/repository"
	httptransport "github.com/denis-sigma/laboratory-work/internal/transport/http"
	"github.com/denis-sigma/laboratory-work/internal/trip"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("загрузка конфигурации: %v", err)
	}

	log.Printf("HTTP сервер адрес: %s", cfg.HTTP.Addr)
	ctx := context.Background()

	pool, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("подключение к бд: %v", err)
	}
	defer pool.Close()

	tripRepository := repository.NewPostgresTripRepository(
		pool,
		cfg.Database.QueryTimeout,
	)

	txManager := database.NewTxManager(pool)
	tripService := trip.NewService(tripRepository, txManager)

	handler := httptransport.NewHandler(
		pool,
		cfg.Database.QueryTimeout,
		tripService,
	)

	server := &http.Server{
		Addr: cfg.HTTP.Addr,
		Handler: api.HandlerWithOptions(
			handler,
			api.ChiServerOptions{
				ErrorHandlerFunc: httptransport.OpenAPIErrorHandler,
			},
		),
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	log.Printf("подключились к БД")

	log.Printf("HTTP сервер слушает на  %s", cfg.HTTP.Addr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("HTTP сервер упал: %v", err)
		}
	}()

	<-stop

	log.Printf("получен сигнал остановки")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("ошибка остановки HTTP-сервера: %v", err)
	}

	log.Printf("HTTP-сервер остановлен")
}
