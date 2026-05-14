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
CREATE TABLE IF NOT EXISTS todos (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  completed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_todos_user_updated_at ON todos (user_id, updated_at DESC);
`
	_, err := pool.Exec(ctx, query)
	return err
}

func (store *poolTodoStore) Ping(ctx context.Context) error {
	return store.pool.Ping(ctx)
}

func (store *poolTodoStore) ListTodos(ctx context.Context, userID string) ([]todo, error) {
	return listTodos(ctx, poolTodoDB{pool: store.pool}, userID)
}

func (store *poolTodoStore) CreateTodo(ctx context.Context, userID string, input todoInput) (todo, error) {
	return createTodo(ctx, poolTodoDB{pool: store.pool}, userID, input)
}

func (store *poolTodoStore) GetTodo(ctx context.Context, id string, userID string) (todo, error) {
	return fetchTodo(ctx, poolTodoDB{pool: store.pool}, id, userID)
}

func (store *poolTodoStore) UpdateTodo(ctx context.Context, id string, userID string, input todoInput) (todo, error) {
	return updateTodo(ctx, poolTodoDB{pool: store.pool}, id, userID, input)
}

func (store *poolTodoStore) DeleteTodo(ctx context.Context, id string, userID string) (bool, error) {
	return deleteTodo(ctx, poolTodoDB{pool: store.pool}, id, userID)
}

func (db poolTodoDB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	ctx, span := startPostgresSpan(ctx, query)
	defer span.End()
	return db.pool.Exec(ctx, query, args...)
}

func (db poolTodoDB) QueryRow(ctx context.Context, query string, args ...any) rowScanner {
	ctx, span := startPostgresSpan(ctx, query)
	return tracedRowScanner{row: db.pool.QueryRow(ctx, query, args...), span: span}
}

func (db poolTodoDB) Query(ctx context.Context, query string, args ...any) (rowIterator, error) {
	ctx, span := startPostgresSpan(ctx, query)
	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		span.End()
		return nil, err
	}
	return &tracedRowIterator{rows: rows, span: span}, nil
}

type tracedRowScanner struct {
	row  rowScanner
	span trace.Span
}

func (scanner tracedRowScanner) Scan(dest ...any) error {
	defer scanner.span.End()
	return scanner.row.Scan(dest...)
}

type tracedRowIterator struct {
	rows rowIterator
	span trace.Span
	done bool
}

func (iterator *tracedRowIterator) Close() {
	iterator.rows.Close()
	if !iterator.done {
		iterator.done = true
		iterator.span.End()
	}
}

func (iterator *tracedRowIterator) Next() bool {
	next := iterator.rows.Next()
	if !next && !iterator.done {
		iterator.done = true
		iterator.span.End()
	}
	return next
}

func (iterator *tracedRowIterator) Scan(dest ...any) error {
	return iterator.rows.Scan(dest...)
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
