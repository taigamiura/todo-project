package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func migrate(ctx context.Context, pool migrator) error {
	query := `
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
	_, err := pool.Exec(ctx, query)
	return err
}

func (store *poolUserStore) Ping(ctx context.Context) error {
	return store.pool.Ping(ctx)
}

func (store *poolUserStore) CreateUser(ctx context.Context, input authInput) (user, error) {
	return createUser(ctx, poolUserDB{pool: store.pool}, input)
}

func (store *poolUserStore) AuthenticateUser(ctx context.Context, input authInput) (user, error) {
	return authenticateUser(ctx, poolUserDB{pool: store.pool}, input)
}

func (db poolUserDB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	ctx, span := startPostgresSpan(ctx, query)
	defer span.End()
	return db.pool.Exec(ctx, query, args...)
}

func (db poolUserDB) QueryRow(ctx context.Context, query string, args ...any) rowScanner {
	ctx, span := startPostgresSpan(ctx, query)
	return tracedRowScanner{row: db.pool.QueryRow(ctx, query, args...), span: span}
}

type tracedRowScanner struct {
	row  rowScanner
	span trace.Span
}

func (scanner tracedRowScanner) Scan(dest ...any) error {
	defer scanner.span.End()
	return scanner.row.Scan(dest...)
}

func startPostgresSpan(ctx context.Context, query string) (context.Context, trace.Span) {
	operation := postgresOperationName(query)
	ctx, span := otel.Tracer(serviceName).Start(ctx, "postgresql "+operation)
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", operation),
	)
	return ctx, span
}

func postgresOperationName(query string) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return "QUERY"
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "QUERY"
	}
	return strings.ToUpper(fields[0])
}
