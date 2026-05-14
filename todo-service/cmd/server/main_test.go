package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeTodoStore struct {
	pingErr  error
	listFn   func(context.Context, string) ([]todo, error)
	createFn func(context.Context, string, todoInput) (todo, error)
	getFn    func(context.Context, string, string) (todo, error)
	updateFn func(context.Context, string, string, todoInput) (todo, error)
	deleteFn func(context.Context, string, string) (bool, error)
}

func (store *fakeTodoStore) Ping(context.Context) error { return store.pingErr }
func (store *fakeTodoStore) ListTodos(ctx context.Context, userID string) ([]todo, error) {
	if store.listFn != nil {
		return store.listFn(ctx, userID)
	}
	return nil, nil
}
func (store *fakeTodoStore) CreateTodo(ctx context.Context, userID string, input todoInput) (todo, error) {
	if store.createFn != nil {
		return store.createFn(ctx, userID, input)
	}
	return todo{}, nil
}
func (store *fakeTodoStore) GetTodo(ctx context.Context, id string, userID string) (todo, error) {
	if store.getFn != nil {
		return store.getFn(ctx, id, userID)
	}
	return todo{}, nil
}
func (store *fakeTodoStore) UpdateTodo(ctx context.Context, id string, userID string, input todoInput) (todo, error) {
	if store.updateFn != nil {
		return store.updateFn(ctx, id, userID, input)
	}
	return todo{}, nil
}
func (store *fakeTodoStore) DeleteTodo(ctx context.Context, id string, userID string) (bool, error) {
	if store.deleteFn != nil {
		return store.deleteFn(ctx, id, userID)
	}
	return false, nil
}

type fakeTodoDB struct {
	execFn     func(context.Context, string, ...any) (pgconn.CommandTag, error)
	queryRowFn func(context.Context, string, ...any) rowScanner
	queryFn    func(context.Context, string, ...any) (rowIterator, error)
}

func (db *fakeTodoDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.execFn(ctx, sql, args...)
}
func (db *fakeTodoDB) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return db.queryRowFn(ctx, sql, args...)
}
func (db *fakeTodoDB) Query(ctx context.Context, sql string, args ...any) (rowIterator, error) {
	return db.queryFn(ctx, sql, args...)
}

type fakeTodoRow struct{ scanFn func(...any) error }

func (row fakeTodoRow) Scan(dest ...any) error { return row.scanFn(dest...) }

type fakeRows struct {
	items   []todo
	index   int
	scanErr error
}

func (rows *fakeRows) Close() {}
func (rows *fakeRows) Next() bool {
	return rows.index < len(rows.items)
}
func (rows *fakeRows) Scan(dest ...any) error {
	if rows.scanErr != nil {
		return rows.scanErr
	}
	item := rows.items[rows.index]
	rows.index++
	*(dest[0].(*string)) = item.ID
	*(dest[1].(*string)) = item.Title
	*(dest[2].(*string)) = item.Description
	*(dest[3].(*bool)) = item.Completed
	*(dest[4].(*time.Time)) = item.CreatedAt
	*(dest[5].(*time.Time)) = item.UpdatedAt
	return nil
}

func TestLoadConfig(t *testing.T) {
	if _, err := loadConfig(func(string) string { return "" }); err == nil {
		t.Fatal("expected error")
	}

	cfg, err := loadConfig(func(key string) string {
		if key == "DATABASE_URL" {
			return "postgres://db"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	if cfg.Port != "8082" {
		t.Fatalf("unexpected port: %s", cfg.Port)
	}
}

func TestBuildRouter(t *testing.T) {
	now := time.Now()
	router := buildRouter(&fakeTodoStore{
		listFn:   func(context.Context, string) ([]todo, error) { return []todo{{ID: "1"}}, nil },
		createFn: func(context.Context, string, todoInput) (todo, error) { return todo{ID: "1"}, nil },
		getFn:    func(context.Context, string, string) (todo, error) { return todo{ID: "1"}, nil },
		updateFn: func(context.Context, string, string, todoInput) (todo, error) {
			return todo{ID: "1", UpdatedAt: now}, nil
		},
		deleteFn: func(context.Context, string, string) (bool, error) { return true, nil },
	})

	for _, tc := range []struct {
		method string
		path   string
		body   string
		userID string
		status int
	}{
		{http.MethodGet, "/healthz", "", "", http.StatusOK},
		{http.MethodGet, "/internal/todos/", "", "u1", http.StatusOK},
		{http.MethodPost, "/internal/todos/", `{"title":"a"}`, "u1", http.StatusCreated},
		{http.MethodGet, "/internal/todos/1/", "", "u1", http.StatusOK},
		{http.MethodPatch, "/internal/todos/1/", `{"title":"a"}`, "u1", http.StatusOK},
		{http.MethodDelete, "/internal/todos/1/", "", "u1", http.StatusNoContent},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.userID != "" {
			req.Header.Set("X-User-ID", tc.userID)
		}
		router.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s %s: got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestBuildRouterErrors(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		userID string
		store  *fakeTodoStore
		status int
	}{
		{"health fail", http.MethodGet, "/healthz", "", "", &fakeTodoStore{pingErr: errors.New("down")}, http.StatusServiceUnavailable},
		{"list missing user", http.MethodGet, "/internal/todos/", "", "", &fakeTodoStore{}, http.StatusUnauthorized},
		{"list store error", http.MethodGet, "/internal/todos/", "", "u1", &fakeTodoStore{listFn: func(context.Context, string) ([]todo, error) { return nil, errors.New("boom") }}, http.StatusInternalServerError},
		{"create bad json", http.MethodPost, "/internal/todos/", "{", "u1", &fakeTodoStore{}, http.StatusBadRequest},
		{"create validation", http.MethodPost, "/internal/todos/", `{"title":""}`, "u1", &fakeTodoStore{}, http.StatusBadRequest},
		{"create error", http.MethodPost, "/internal/todos/", `{"title":"a"}`, "u1", &fakeTodoStore{createFn: func(context.Context, string, todoInput) (todo, error) { return todo{}, errors.New("boom") }}, http.StatusInternalServerError},
		{"get missing", http.MethodGet, "/internal/todos/1/", "", "u1", &fakeTodoStore{getFn: func(context.Context, string, string) (todo, error) { return todo{}, errNotFound }}, http.StatusNotFound},
		{"get internal", http.MethodGet, "/internal/todos/1/", "", "u1", &fakeTodoStore{getFn: func(context.Context, string, string) (todo, error) { return todo{}, errors.New("boom") }}, http.StatusInternalServerError},
		{"patch missing user", http.MethodPatch, "/internal/todos/1/", `{"title":"a"}`, "", &fakeTodoStore{}, http.StatusUnauthorized},
		{"patch bad json", http.MethodPatch, "/internal/todos/1/", "{", "u1", &fakeTodoStore{}, http.StatusBadRequest},
		{"patch validation", http.MethodPatch, "/internal/todos/1/", `{"title":""}`, "u1", &fakeTodoStore{}, http.StatusBadRequest},
		{"patch error", http.MethodPatch, "/internal/todos/1/", `{"title":"a"}`, "u1", &fakeTodoStore{updateFn: func(context.Context, string, string, todoInput) (todo, error) { return todo{}, errors.New("boom") }}, http.StatusInternalServerError},
		{"delete error", http.MethodDelete, "/internal/todos/1/", "", "u1", &fakeTodoStore{deleteFn: func(context.Context, string, string) (bool, error) { return false, errors.New("boom") }}, http.StatusInternalServerError},
		{"delete not found", http.MethodDelete, "/internal/todos/1/", "", "u1", &fakeTodoStore{deleteFn: func(context.Context, string, string) (bool, error) { return false, nil }}, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			buildRouter(tt.store).ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("unexpected status: %d", rec.Code)
			}
		})
	}
}

func TestMigrateAndHelpers(t *testing.T) {
	errExpected := errors.New("exec failed")
	err := migrate(context.Background(), &fakeTodoDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag(""), errExpected
	}})
	if err != errExpected {
		t.Fatalf("unexpected error: %v", err)
	}

	err = migrate(context.Background(), &fakeTodoDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag("CREATE TABLE"), nil
	}})
	if err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	if _, err := fetchTodo(context.Background(), &fakeTodoDB{queryRowFn: func(context.Context, string, ...any) rowScanner {
		return fakeTodoRow{scanFn: func(...any) error { return errors.New("missing") }}
	}}, "1", " u1 "); !errors.Is(err, errNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}

	item, err := fetchTodo(context.Background(), &fakeTodoDB{queryRowFn: func(_ context.Context, _ string, args ...any) rowScanner {
		if args[1] != "u1" {
			t.Fatalf("userID not trimmed: %#v", args)
		}
		return fakeTodoRow{scanFn: func(dest ...any) error {
			*(dest[0].(*string)) = "1"
			*(dest[1].(*string)) = "title"
			*(dest[2].(*string)) = "desc"
			*(dest[3].(*bool)) = true
			*(dest[4].(*time.Time)) = nowTime()
			*(dest[5].(*time.Time)) = nowTime()
			return nil
		}}
	}}, "1", " u1 ")
	if err != nil || item.ID != "1" {
		t.Fatalf("unexpected result: %#v %v", item, err)
	}

	if _, err := listTodos(context.Background(), &fakeTodoDB{queryFn: func(context.Context, string, ...any) (rowIterator, error) {
		return nil, errors.New("boom")
	}}, "u1"); err == nil {
		t.Fatal("expected error")
	}

	if _, err := listTodos(context.Background(), &fakeTodoDB{queryFn: func(context.Context, string, ...any) (rowIterator, error) {
		return &fakeRows{items: []todo{{ID: "1"}}, scanErr: errors.New("scan")}, nil
	}}, "u1"); err == nil {
		t.Fatal("expected scan error")
	}

	todos, err := listTodos(context.Background(), &fakeTodoDB{queryFn: func(context.Context, string, ...any) (rowIterator, error) {
		return &fakeRows{items: []todo{{ID: "1", Title: "a", CreatedAt: nowTime(), UpdatedAt: nowTime()}}}, nil
	}}, "u1")
	if err != nil || len(todos) != 1 {
		t.Fatalf("unexpected result: %#v %v", todos, err)
	}

	if _, err := createTodo(context.Background(), &fakeTodoDB{queryRowFn: func(context.Context, string, ...any) rowScanner {
		return fakeTodoRow{scanFn: func(...any) error { return errors.New("boom") }}
	}}, "u1", todoInput{Title: " a ", Description: " b "}); err == nil {
		t.Fatal("expected error")
	}

	created, err := createTodo(context.Background(), &fakeTodoDB{queryRowFn: func(_ context.Context, _ string, args ...any) rowScanner {
		if args[2] != "a" || args[3] != "b" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return fakeTodoRow{scanFn: func(dest ...any) error {
			*(dest[0].(*time.Time)) = nowTime()
			*(dest[1].(*time.Time)) = nowTime()
			return nil
		}}
	}}, "u1", todoInput{Title: " a ", Description: " b "})
	if err != nil || created.Title != "a" || created.Description != "b" {
		t.Fatalf("unexpected create: %#v %v", created, err)
	}

	if _, err := updateTodo(context.Background(), &fakeTodoDB{queryRowFn: func(context.Context, string, ...any) rowScanner {
		return fakeTodoRow{scanFn: func(...any) error { return errors.New("boom") }}
	}}, "1", "u1", todoInput{Title: "a"}); err == nil {
		t.Fatal("expected error")
	}

	updated, err := updateTodo(context.Background(), &fakeTodoDB{queryRowFn: func(_ context.Context, _ string, args ...any) rowScanner {
		if args[0] != "a" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return fakeTodoRow{scanFn: func(dest ...any) error {
			*(dest[0].(*string)) = "1"
			*(dest[1].(*string)) = "a"
			*(dest[2].(*string)) = ""
			*(dest[3].(*bool)) = false
			*(dest[4].(*time.Time)) = nowTime()
			*(dest[5].(*time.Time)) = nowTime()
			return nil
		}}
	}}, "1", "u1", todoInput{Title: " a "})
	if err != nil || updated.Title != "a" {
		t.Fatalf("unexpected update: %#v %v", updated, err)
	}

	if _, err := deleteTodo(context.Background(), &fakeTodoDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag(""), errors.New("boom")
	}}, "1", "u1"); err == nil {
		t.Fatal("expected error")
	}

	deleted, err := deleteTodo(context.Background(), &fakeTodoDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag("DELETE 0"), nil
	}}, "1", "u1")
	if err != nil || deleted {
		t.Fatalf("unexpected delete result: %v %v", deleted, err)
	}

	deleted, err = deleteTodo(context.Background(), &fakeTodoDB{execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag("DELETE 1"), nil
	}}, "1", "u1")
	if err != nil || !deleted {
		t.Fatalf("unexpected delete result: %v %v", deleted, err)
	}
}

func TestValidateTodoInput(t *testing.T) {
	for _, tc := range []struct {
		input todoInput
		err   string
	}{
		{todoInput{Title: ""}, "タイトルは必須です。"},
		{todoInput{Title: strings.Repeat("a", 51)}, "タイトルは50文字以内で入力してください。"},
		{todoInput{Title: "a", Description: strings.Repeat("b", 301)}, "説明は300文字以内で入力してください。"},
	} {
		if err := validateTodoInput(tc.input); err == nil || err.Error() != tc.err {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if err := validateTodoInput(todoInput{Title: "ok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func nowTime() time.Time {
	return time.Unix(1, 0)
}

func TestPoolAdapters(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://127.0.0.1:1/test?sslmode=disable")
	if err != nil {
		t.Fatalf("pgxpool.New error: %v", err)
	}
	defer pool.Close()

	store := &poolTodoStore{pool: pool}
	if err := store.Ping(context.Background()); err == nil {
		t.Fatal("expected ping error")
	}
	if _, err := store.ListTodos(context.Background(), "u1"); err == nil {
		t.Fatal("expected list error")
	}
	if _, err := store.CreateTodo(context.Background(), "u1", todoInput{Title: "a"}); err == nil {
		t.Fatal("expected create error")
	}
	if _, err := store.GetTodo(context.Background(), "1", "u1"); err == nil {
		t.Fatal("expected get error")
	}
	if _, err := store.UpdateTodo(context.Background(), "1", "u1", todoInput{Title: "a"}); err == nil {
		t.Fatal("expected update error")
	}
	if _, err := store.DeleteTodo(context.Background(), "1", "u1"); err == nil {
		t.Fatal("expected delete error")
	}

	adapter := poolTodoDB{pool: pool}
	if _, err := adapter.Exec(context.Background(), "SELECT 1"); err == nil {
		t.Fatal("expected exec error")
	}
	if err := adapter.QueryRow(context.Background(), "SELECT 1").Scan(new(int)); err == nil {
		t.Fatal("expected row error")
	}
	if _, err := adapter.Query(context.Background(), "SELECT 1"); err == nil {
		t.Fatal("expected query error")
	}
}

func TestMainMissingConfig(t *testing.T) {
	if os.Getenv("GO_WANT_TODO_MAIN") == "1" {
		_ = os.Unsetenv("DATABASE_URL")
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainMissingConfig")
	cmd.Env = append(os.Environ(), "GO_WANT_TODO_MAIN=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
}
