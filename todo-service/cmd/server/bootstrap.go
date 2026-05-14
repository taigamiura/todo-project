package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, stop := setupRuntimeContext()
	defer stop()

	telemetryShutdown, err := setupTelemetry(ctx, serviceName)
	if err != nil {
		appLogger.Error("telemetry_init_failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = telemetryShutdown(shutdownCtx)
	}()

	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		appLogger.Error("config_load_failed", "error", err)
		os.Exit(1)
	}

	pool, err := openDatabasePool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		appLogger.Error("database_connect_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migrate(context.Background(), pool); err != nil {
		appLogger.Error("migration_failed", "error", err)
		os.Exit(1)
	}

	server := newHTTPServer(cfg.Port, buildRouter(&poolTodoStore{pool: pool}))

	appLogger.Info("server_start", "addr", server.Addr)
	if err := runHTTPServer(ctx, server, appLogger); err != nil {
		appLogger.Error("server_failed", "error", err)
		os.Exit(1)
	}
}

func loadConfig(getenv func(string) string) (appConfig, error) {
	databaseURL := getenv("DATABASE_URL")
	if databaseURL == "" {
		return appConfig{}, fmt.Errorf("DATABASE_URL is required")
	}

	port := getenv("PORT")
	if port == "" {
		port = "8082"
	}

	return appConfig{DatabaseURL: databaseURL, Port: port}, nil
}

func openDatabasePool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTrimSQLInSpanName(),
		otelpgx.WithDisableConnectionDetailsInAttributes(),
	)
	poolConfig.MaxConns = 50
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return pool, nil
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}
