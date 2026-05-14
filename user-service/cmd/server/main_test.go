package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserStore struct {
	pingErr      error
	createFn     func(context.Context, authInput) (user, error)
	authFn       func(context.Context, authInput) (user, error)
	createCalled bool
	authCalled   bool
}

func (store *fakeUserStore) Ping(context.Context) error {
	return store.pingErr
}

func (store *fakeUserStore) CreateUser(ctx context.Context, input authInput) (user, error) {
	store.createCalled = true
	if store.createFn != nil {
		return store.createFn(ctx, input)
	}
	return user{}, nil
}

func (store *fakeUserStore) AuthenticateUser(ctx context.Context, input authInput) (user, error) {
	store.authCalled = true
	if store.authFn != nil {
		return store.authFn(ctx, input)
	}
	return user{}, nil
}

type fakeUserDB struct {
	execFn     func(context.Context, string, ...any) (pgconn.CommandTag, error)
	queryRowFn func(context.Context, string, ...any) rowScanner
}

func (db *fakeUserDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.execFn(ctx, sql, args...)
}

func (db *fakeUserDB) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return db.queryRowFn(ctx, sql, args...)
}

type fakeRow struct {
	scanFn func(...any) error
}

func (row fakeRow) Scan(dest ...any) error {
	return row.scanFn(dest...)
}

func TestLoadConfig(t *testing.T) {
	t.Run("missing database url", func(t *testing.T) {
		_, err := loadConfig(func(string) string { return "" })
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("uses default port", func(t *testing.T) {
		cfg, err := loadConfig(func(key string) string {
			if key == "DATABASE_URL" {
				return "postgres://db"
			}
			return ""
		})
		if err != nil {
			t.Fatalf("loadConfig error: %v", err)
		}
		if cfg.Port != "8081" {
			t.Fatalf("unexpected port: %s", cfg.Port)
		}
	})
}

func TestBuildRouterHealth(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

		buildRouter(&fakeUserStore{}).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
	})

	t.Run("failure", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

		buildRouter(&fakeUserStore{pingErr: errors.New("down")}).ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
	})
}

func TestBuildRouterSignup(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		store  *fakeUserStore
		status int
	}{
		{name: "invalid json", body: "{", store: &fakeUserStore{}, status: http.StatusBadRequest},
		{name: "conflict", body: `{"name":"a","email":"a@example.com","password":"password"}`, store: &fakeUserStore{createFn: func(context.Context, authInput) (user, error) { return user{}, errConflict }}, status: http.StatusConflict},
		{name: "validation", body: `{"name":"a","email":"a@example.com","password":"password"}`, store: &fakeUserStore{createFn: func(context.Context, authInput) (user, error) { return user{}, errValidation }}, status: http.StatusBadRequest},
		{name: "internal", body: `{"name":"a","email":"a@example.com","password":"password"}`, store: &fakeUserStore{createFn: func(context.Context, authInput) (user, error) { return user{}, errors.New("boom") }}, status: http.StatusInternalServerError},
		{name: "success", body: `{"name":"a","email":"a@example.com","password":"password"}`, store: &fakeUserStore{createFn: func(_ context.Context, input authInput) (user, error) {
			return user{ID: "1", Name: input.Name, Email: input.Email}, nil
		}}, status: http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/internal/users/signup", strings.NewReader(tt.body))

			buildRouter(tt.store).ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("unexpected status: %d", rec.Code)
			}
		})
	}
}

func TestBuildRouterLogin(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		store  *fakeUserStore
		status int
	}{
		{name: "invalid json", body: "{", store: &fakeUserStore{}, status: http.StatusBadRequest},
		{name: "unauthorized", body: `{"email":"a@example.com","password":"password"}`, store: &fakeUserStore{authFn: func(context.Context, authInput) (user, error) { return user{}, errUnauthorized }}, status: http.StatusUnauthorized},
		{name: "validation", body: `{"email":"a@example.com","password":"password"}`, store: &fakeUserStore{authFn: func(context.Context, authInput) (user, error) { return user{}, errValidation }}, status: http.StatusBadRequest},
		{name: "internal", body: `{"email":"a@example.com","password":"password"}`, store: &fakeUserStore{authFn: func(context.Context, authInput) (user, error) { return user{}, errors.New("boom") }}, status: http.StatusInternalServerError},
		{name: "success", body: `{"email":"a@example.com","password":"password"}`, store: &fakeUserStore{authFn: func(_ context.Context, input authInput) (user, error) { return user{ID: "1", Email: input.Email}, nil }}, status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/internal/users/login", strings.NewReader(tt.body))

			buildRouter(tt.store).ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("unexpected status: %d", rec.Code)
			}
		})
	}
}

func TestMigrate(t *testing.T) {
	errExpected := errors.New("exec failed")
	_, err := (&fakeUserDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag(""), errExpected
	}}).Exec(context.Background(), "")
	if err != errExpected {
		t.Fatalf("unexpected exec error: %v", err)
	}

	err = migrate(context.Background(), &fakeUserDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag("CREATE TABLE"), nil
	}})
	if err != nil {
		t.Fatalf("migrate error: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		_, err := createUser(context.Background(), &fakeUserDB{}, authInput{})
		if !errors.Is(err, errValidation) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		_, err := createUser(context.Background(), &fakeUserDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag(""), errors.New("duplicate key")
		}}, authInput{Name: " A ", Email: "A@EXAMPLE.COM", Password: "password"})
		if !errors.Is(err, errConflict) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("generic exec error", func(t *testing.T) {
		_, err := createUser(context.Background(), &fakeUserDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag(""), errors.New("boom")
		}}, authInput{Name: " A ", Email: "A@EXAMPLE.COM", Password: "password"})
		if err == nil || err.Error() != "boom" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		var captured []any
		created, err := createUser(context.Background(), &fakeUserDB{execFn: func(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
			captured = args
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		}}, authInput{Name: " A ", Email: "A@EXAMPLE.COM", Password: "password"})
		if err != nil {
			t.Fatalf("createUser error: %v", err)
		}
		if created.Name != "A" || created.Email != "a@example.com" {
			t.Fatalf("unexpected user: %#v", created)
		}
		if captured[1] != "A" || captured[2] != "a@example.com" {
			t.Fatalf("unexpected args: %#v", captured)
		}
	})
}

func TestAuthenticateUser(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		_, err := authenticateUser(context.Background(), &fakeUserDB{}, authInput{})
		if !errors.Is(err, errValidation) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := authenticateUser(context.Background(), &fakeUserDB{queryRowFn: func(context.Context, string, ...any) rowScanner {
			return fakeRow{scanFn: func(...any) error { return errors.New("missing") }}
		}}, authInput{Email: "a@example.com", Password: "password"})
		if !errors.Is(err, errUnauthorized) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		_, err := authenticateUser(context.Background(), &fakeUserDB{queryRowFn: func(context.Context, string, ...any) rowScanner {
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*string)) = "1"
				*(dest[1].(*string)) = "name"
				*(dest[2].(*string)) = "a@example.com"
				*(dest[3].(*string)) = string(hash)
				return nil
			}}
		}}, authInput{Email: "a@example.com", Password: "different"})
		if !errors.Is(err, errUnauthorized) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		found, err := authenticateUser(context.Background(), &fakeUserDB{queryRowFn: func(context.Context, string, ...any) rowScanner {
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*string)) = "1"
				*(dest[1].(*string)) = "name"
				*(dest[2].(*string)) = "a@example.com"
				*(dest[3].(*string)) = string(hash)
				return nil
			}}
		}}, authInput{Email: "A@EXAMPLE.COM", Password: "password"})
		if err != nil {
			t.Fatalf("authenticateUser error: %v", err)
		}
		if found.Email != "a@example.com" {
			t.Fatalf("unexpected user: %#v", found)
		}
	})
}

func TestNormalizeAuthInput(t *testing.T) {
	tests := []struct {
		name        string
		input       authInput
		requireName bool
		error       error
	}{
		{name: "missing name", input: authInput{Email: "a@example.com", Password: "password"}, requireName: true, error: errValidation},
		{name: "bad email", input: authInput{Name: "a", Email: "bad", Password: "password"}, requireName: true, error: errValidation},
		{name: "short password", input: authInput{Name: "a", Email: "a@example.com", Password: "short"}, requireName: true, error: errValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeAuthInput(tt.input, tt.requireName)
			if !errors.Is(err, tt.error) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}

	normalized, err := normalizeAuthInput(authInput{Name: " A ", Email: "A@EXAMPLE.COM", Password: " password "}, true)
	if err != nil {
		t.Fatalf("normalizeAuthInput error: %v", err)
	}
	if normalized.Name != "A" || normalized.Email != "a@example.com" || normalized.Password != "password" {
		t.Fatalf("unexpected normalized input: %#v", normalized)
	}
}

func TestWriteHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusAccepted, map[string]string{"ok": "1"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "bad")
	var payload errorResponse
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if payload.Error.Message != "bad" || payload.Error.Code != errorCodeBadRequest || payload.Error.Status != http.StatusBadRequest {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestPoolAdapters(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://127.0.0.1:1/test?sslmode=disable")
	if err != nil {
		t.Fatalf("pgxpool.New error: %v", err)
	}
	defer pool.Close()

	store := &poolUserStore{pool: pool}
	if err := store.Ping(context.Background()); err == nil {
		t.Fatal("expected ping error")
	}
	if _, err := store.CreateUser(context.Background(), authInput{Name: "a", Email: "a@example.com", Password: "password"}); err == nil {
		t.Fatal("expected create error")
	}
	if _, err := store.AuthenticateUser(context.Background(), authInput{Email: "a@example.com", Password: "password"}); err == nil {
		t.Fatal("expected auth error")
	}

	adapter := poolUserDB{pool: pool}
	if _, err := adapter.Exec(context.Background(), "SELECT 1"); err == nil {
		t.Fatal("expected exec error")
	}
	if err := adapter.QueryRow(context.Background(), "SELECT 1").Scan(new(int)); err == nil {
		t.Fatal("expected query row error")
	}
}

func TestMainMissingConfig(t *testing.T) {
	if os.Getenv("GO_WANT_USER_MAIN") == "1" {
		_ = os.Unsetenv("DATABASE_URL")
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainMissingConfig")
	cmd.Env = append(os.Environ(), "GO_WANT_USER_MAIN=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
}
