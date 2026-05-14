package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type todo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type todoInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type appConfig struct {
	DatabaseURL string
	Port        string
}

type todoStore interface {
	Ping(context.Context) error
	ListTodos(context.Context, string) ([]todo, error)
	CreateTodo(context.Context, string, todoInput) (todo, error)
	GetTodo(context.Context, string, string) (todo, error)
	UpdateTodo(context.Context, string, string, todoInput) (todo, error)
	DeleteTodo(context.Context, string, string) (bool, error)
}

type todoDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) rowScanner
	Query(context.Context, string, ...any) (rowIterator, error)
}

type migrator interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type rowScanner interface {
	Scan(...any) error
}

type rowIterator interface {
	Close()
	Next() bool
	Scan(...any) error
}

type poolTodoStore struct {
	pool *pgxpool.Pool
}

type poolTodoDB struct {
	pool *pgxpool.Pool
}
