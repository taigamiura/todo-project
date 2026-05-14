package main

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

var (
	errConflict     = errors.New("conflict")
	errUnauthorized = errors.New("unauthorized")
	errValidation   = errors.New("validation")
)

func createUser(ctx context.Context, pool userDB, input authInput) (user, error) {
	normalized, err := normalizeAuthInput(input, true)
	if err != nil {
		return user{}, err
	}

	_, hashSpan := otel.Tracer(serviceName).Start(ctx, "auth.hash_password")
	hashSpan.SetAttributes(attribute.Bool("user.signup", true))
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(normalized.Password), bcrypt.DefaultCost)
	hashSpan.End()
	if err != nil {
		return user{}, err
	}

	created := user{
		ID:    uuid.NewString(),
		Name:  normalized.Name,
		Email: normalized.Email,
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)`,
		created.ID,
		created.Name,
		created.Email,
		string(passwordHash),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return user{}, errConflict
		}
		return user{}, err
	}

	return created, nil
}

func authenticateUser(ctx context.Context, pool userDB, input authInput) (user, error) {
	normalized, err := normalizeAuthInput(input, false)
	if err != nil {
		return user{}, err
	}

	var found storedUser
	err = pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash FROM users WHERE email = $1`,
		normalized.Email,
	).Scan(&found.ID, &found.Name, &found.Email, &found.PasswordHash)
	if err != nil {
		return user{}, errUnauthorized
	}

	_, verifySpan := otel.Tracer(serviceName).Start(ctx, "auth.verify_password")
	verifySpan.SetAttributes(attribute.Bool("user.login", true))
	if err := bcrypt.CompareHashAndPassword([]byte(found.PasswordHash), []byte(normalized.Password)); err != nil {
		verifySpan.End()
		return user{}, errUnauthorized
	}
	verifySpan.End()

	return found.user, nil
}

func normalizeAuthInput(input authInput, requireName bool) (authInput, error) {
	normalized := authInput{
		Name:     strings.TrimSpace(input.Name),
		Email:    strings.ToLower(strings.TrimSpace(input.Email)),
		Password: strings.TrimSpace(input.Password),
	}

	if requireName && normalized.Name == "" {
		return authInput{}, errValidation
	}
	if normalized.Email == "" || !strings.Contains(normalized.Email, "@") {
		return authInput{}, errValidation
	}
	if len(normalized.Password) < 8 {
		return authInput{}, errValidation
	}

	return normalized, nil
}
