package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type user struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type storedUser struct {
	user
	PasswordHash string
}

type authInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type appConfig struct {
	DatabaseURL string
	Port        string
}

type userStore interface {
	Ping(context.Context) error
	CreateUser(context.Context, authInput) (user, error)
	AuthenticateUser(context.Context, authInput) (user, error)
}

type userDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) rowScanner
}

type migrator interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type rowScanner interface {
	Scan(...any) error
}

type poolUserStore struct {
	pool *pgxpool.Pool
}

type poolUserDB struct {
	pool *pgxpool.Pool
}
