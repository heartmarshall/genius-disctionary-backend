package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/heartmarshall/my-english/internal/app"
	"github.com/heartmarshall/my-english/internal/config"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/service"
	transport "github.com/heartmarshall/my-english/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Загрузка конфигурации
	cfg, err := config.Load(".env") // или путь к config.yaml
	if err != nil {
		// Если конфиг не загрузился, мы не можем даже логировать нормально
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 2. Инициализация логгера
	logger := app.NewLogger(cfg.Log)
	// Устанавливаем как дефолтный, чтобы библиотеки тоже писали туда
	slog.SetDefault(logger)

	logger.Info("starting application", slog.String("env", "dev")) // Можно добавить поле env в конфиг

	// 3. Подключение к БД
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to parse db config", slog.Any("error", err))
		os.Exit(1)
	}

	// Настройка пула
	poolConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.Database.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	// Проверка соединения
	if err := pool.Ping(context.Background()); err != nil {
		logger.Error("failed to ping database", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("connected to database")

	// 4. Инициализация слоев
	// Repository
	repos := repository.NewRegistry(pool)
	txManager := database.NewTxManager(pool)

	// Services
	// TODO: Сюда нужно будет добавить реальных провайдеров для SuggestionService (OpenAI, FreeDict)
	// Пока передаем пустой список или nil
	services, err := service.NewServices(service.Deps{
		Repos:     repos,
		TxManager: txManager,
		Providers: nil, // Провайдеры подсказок добавим позже
	})
	if err != nil {
		logger.Error("failed to initialize services", slog.Any("error", err))
		os.Exit(1)
	}

	// 5. Настройка HTTP сервера
	handler := transport.NewHandler(cfg, logger, services, repos)

	// Добавляем Health Check отдельно, чтобы он не проходил через все middleware (опционально)
	healthChecker := app.NewHealthChecker(pool)
	mux := http.NewServeMux()
	mux.Handle("/health", healthChecker.Handler())
	mux.Handle("/ready", healthChecker.ReadinessHandler())
	mux.Handle("/", handler) // Всё остальное идет в основной хендлер

	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Запуск сервера (Graceful Shutdown)
	go func() {
		logger.Info("server starting", slog.String("addr", cfg.Server.Addr()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("server exited properly")
}
