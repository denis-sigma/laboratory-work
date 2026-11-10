package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/denis-sigma/laboratory-work/internal/config"
	"github.com/denis-sigma/laboratory-work/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("загрузка конфигурации: %v", err)
	}

	ctx := context.Background()

	app, err := container.New(ctx, cfg)
	if err != nil {
		log.Fatalf("сборка приложения: %v", err)
	}
	defer app.Pool.Close()

	defer func() {
		if err := app.TracerShutdown(context.Background()); err != nil {
			app.Logger.Error("ошибка остановки OpenTelemetry", "error", err)
		}
	}()

	defer func() {
		if err := app.MeterShutdown(context.Background()); err != nil {
			app.Logger.Error("ошибка остановки OpenTelemetry metrics", "error", err)
		}
	}()

	app.Logger.Info("HTTP сервер запускается", "addr", cfg.HTTP.Addr)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           app.Handler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	app.Logger.Info("подключились к БД")

	app.Logger.Info("HTTP сервер слушает", "addr", cfg.HTTP.Addr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("HTTP сервер упал: %v", err)
		}
	}()

	<-stop

	app.Logger.Info("получен сигнал остановки")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		app.Logger.Error("ошибка остановки HTTP-сервера", "error", err)
		return
	}

	app.Logger.Info("HTTP-сервер остановлен")
}
