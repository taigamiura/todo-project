package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type fakeCache struct {
	pingErr error
	values  map[string]string
	setKeys []string
	delKeys []string
}

func (cache *fakeCache) Ping(context.Context) error { return cache.pingErr }
func (cache *fakeCache) Get(_ context.Context, key string) (string, error) {
	value, ok := cache.values[key]
	if !ok {
		return "", errors.New("missing")
	}
	return value, nil
}
func (cache *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if cache.values == nil {
		cache.values = map[string]string{}
	}
	cache.values[key] = string(value)
	cache.setKeys = append(cache.setKeys, key)
	return nil
}
func (cache *fakeCache) Del(_ context.Context, keys ...string) error {
	cache.delKeys = append(cache.delKeys, keys...)
	return nil
}

type fakeDoer struct {
	doFn func(*http.Request) (*http.Response, error)
}

func (doer fakeDoer) Do(req *http.Request) (*http.Response, error) {
	return doer.doFn(req)
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read error") }
func (errReadCloser) Close() error             { return nil }

func TestLoadConfig(t *testing.T) {
	if _, err := loadConfig(func(string) string { return "" }); err == nil {
		t.Fatal("expected error")
	}

	cfg, err := loadConfig(func(key string) string {
		switch key {
		case "USER_SERVICE_URL":
			return "http://user"
		case "TODO_SERVICE_URL":
			return "http://todo"
		case "APP_SESSION_SECRET":
			return "secret"
		case "CACHE_TTL":
			return "bad"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	if cfg.Port != "8080" || cfg.RedisAddr != "redis:6379" || cfg.CacheTTL != 30*time.Second {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestDefaultString(t *testing.T) {
	if defaultString("", "fallback") != "fallback" || defaultString("value", "fallback") != "value" {
		t.Fatal("defaultString failed")
	}
}

func TestForwardAuth(t *testing.T) {
	if _, err := forwardAuth(context.Background(), fakeDoer{}, ":", nil); err == nil {
		t.Fatal("expected invalid url error")
	}

	if _, err := forwardAuth(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	}}, "http://example.com", nil); err == nil {
		t.Fatal("expected client error")
	}

	if _, err := forwardAuth(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{}}, nil
	}}, "http://example.com", nil); err == nil {
		t.Fatal("expected read error")
	}

	if _, err := forwardAuth(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader(`{"error":"exists"}`))}, nil
	}}, "http://example.com", nil); err == nil || err.Error() != "exists" {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := forwardAuth(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader(`oops`))}, nil
	}}, "http://example.com", nil); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := forwardAuth(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{`))}, nil
	}}, "http://example.com", nil); err == nil {
		t.Fatal("expected unmarshal error")
	}

	found, err := forwardAuth(context.Background(), fakeDoer{doFn: func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.Header.Get("Content-Type") != "application/json" {
			t.Fatal("unexpected request")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"user":{"id":"1","email":"a@example.com"}}`))}, nil
	}}, "http://example.com", strings.NewReader(`{}`))
	if err != nil || found.ID != "1" {
		t.Fatalf("unexpected result: %#v %v", found, err)
	}
}

func TestSignTokenAndAuthMiddleware(t *testing.T) {
	token, err := signToken("secret", user{ID: "1", Name: "name", Email: "a@example.com"})
	if err != nil {
		t.Fatalf("signToken error: %v", err)
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return []byte("secret"), nil
	})
	if err != nil || !parsed.Valid || claims["sub"] != "1" {
		t.Fatalf("unexpected token parse: %v %#v", err, claims)
	}

	handler := authMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentUser := r.Context().Value(userContextKey).(user)
		fmt.Fprint(w, currentUser.ID)
	}))

	for _, tc := range []struct {
		name   string
		header string
		status int
		body   string
	}{
		{"missing", "", http.StatusUnauthorized, ""},
		{"bad", "Bearer bad-token", http.StatusUnauthorized, ""},
		{"wrong method", "Bearer " + mustSignMethodToken(t), http.StatusUnauthorized, ""},
		{"missing claims", "Bearer " + mustSignClaimsToken(t, jwt.MapClaims{"sub": "", "email": ""}), http.StatusUnauthorized, ""},
		{"success", "Bearer " + token, http.StatusOK, "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("unexpected status: %d", rec.Code)
			}
			if tc.body != "" && rec.Body.String() != tc.body {
				t.Fatalf("unexpected body: %s", rec.Body.String())
			}
		})
	}
}

func mustSignMethodToken(t *testing.T) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{"sub": "1", "email": "a@example.com"})
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	return signed
}

func mustSignClaimsToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	return signed
}

func TestProxyTodoRequest(t *testing.T) {
	if _, status, err := proxyTodoRequest(context.Background(), fakeDoer{}, ":", "u1", http.MethodGet, nil); err == nil || status != http.StatusInternalServerError {
		t.Fatalf("unexpected result: %d %v", status, err)
	}

	if _, status, err := proxyTodoRequest(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	}}, "http://todo", "u1", http.MethodGet, nil); err == nil || status != http.StatusBadGateway {
		t.Fatalf("unexpected result: %d %v", status, err)
	}

	if _, status, err := proxyTodoRequest(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{}}, nil
	}}, "http://todo", "u1", http.MethodGet, nil); err == nil || status != http.StatusBadGateway {
		t.Fatalf("unexpected result: %d %v", status, err)
	}

	if _, status, err := proxyTodoRequest(context.Background(), fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":"missing"}`))}, nil
	}}, "http://todo", "u1", http.MethodGet, nil); err == nil || status != http.StatusNotFound {
		t.Fatalf("unexpected result: %d %v", status, err)
	}

	payload, status, err := proxyTodoRequest(context.Background(), fakeDoer{doFn: func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("X-User-ID") != "u1" {
			t.Fatal("missing user header")
		}
		if req.Method == http.MethodPatch && req.Header.Get("Content-Type") != "application/json" {
			t.Fatal("missing content type")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"todos":[]}`))}, nil
	}}, "http://todo", "u1", http.MethodPatch, strings.NewReader(`{}`))
	if err != nil || status != http.StatusOK || string(payload) != `{"todos":[]}` {
		t.Fatalf("unexpected result: %d %v %s", status, err, string(payload))
	}
}

func TestDecodeForwardErrorAndInvalidateCache(t *testing.T) {
	if err := decodeForwardError([]byte(`{"error":{"code":"BAD_REQUEST","message":"bad","status":400}}`), 400); err.Error() != "bad" {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := decodeForwardError([]byte(`{"error":"legacy bad"}`), 400); err.Error() != "legacy bad" {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := decodeForwardError([]byte(`oops`), 500); !strings.Contains(err.Error(), "500") {
		t.Fatalf("unexpected error: %v", err)
	}

	cache := &fakeCache{}
	invalidateTodoCache(context.Background(), cache, "u1", "t1")
	if len(cache.delKeys) != 2 || cache.delKeys[0] != "todos:u1" || cache.delKeys[1] != "todo:u1:t1" {
		t.Fatalf("unexpected keys: %#v", cache.delKeys)
	}

	cache = &fakeCache{}
	invalidateTodoCache(context.Background(), cache, "u1", "")
	if len(cache.delKeys) != 1 || cache.delKeys[0] != "todos:u1" {
		t.Fatalf("unexpected keys: %#v", cache.delKeys)
	}
}

func TestWriteHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusAccepted, map[string]string{"ok": "1"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	writeRawJSON(rec, http.StatusCreated, []byte(`{"ok":true}`))
	if rec.Body.String() != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "bad")
	if !strings.Contains(rec.Body.String(), "bad") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	for _, tc := range []struct {
		err    error
		status int
	}{
		{errors.New("すでに登録されています"), http.StatusConflict},
		{errors.New("正しくありません"), http.StatusUnauthorized},
		{errors.New("必須です"), http.StatusBadRequest},
		{errors.New("other"), http.StatusBadGateway},
	} {
		rec = httptest.NewRecorder()
		writeForwardError(rec, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

func TestBuildRouter(t *testing.T) {
	cache := &fakeCache{values: map[string]string{
		"todos:u1":   `{"todos":[{"id":"cached"}]}`,
		"todo:u1:t1": `{"todo":{"id":"cached"}}`,
	}}
	requests := 0
	httpClient := fakeDoer{doFn: func(req *http.Request) (*http.Response, error) {
		requests++
		switch {
		case strings.Contains(req.URL.Path, "/internal/users/signup"):
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"user":{"id":"u1","name":"user","email":"a@example.com"}}`))}, nil
		case strings.Contains(req.URL.Path, "/internal/users/login"):
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"user":{"id":"u1","name":"user","email":"a@example.com"}}`))}, nil
		case req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/internal/todos/t2/"):
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"todo":{"id":"t2"}}`))}, nil
		case req.Method == http.MethodGet:
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"todos":[{"id":"t1"}]}`))}, nil
		case req.Method == http.MethodPost:
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"todo":{"id":"t3"}}`))}, nil
		case req.Method == http.MethodPatch:
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"todo":{"id":"t2"}}`))}, nil
		default:
			return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(bytes.NewReader(nil))}, nil
		}
	}}

	router := buildRouter(config{UserServiceURL: "http://user", TodoServiceURL: "http://todo", AppSessionSecret: "secret", CacheTTL: time.Second}, cache, httpClient)
	token, _ := signToken("secret", user{ID: "u1", Name: "user", Email: "a@example.com"})

	for _, tc := range []struct {
		method string
		path   string
		body   string
		auth   bool
		status int
	}{
		{http.MethodGet, "/healthz", "", false, http.StatusOK},
		{http.MethodPost, "/v1/auth/signup", `{}`, false, http.StatusCreated},
		{http.MethodPost, "/v1/auth/login", `{}`, false, http.StatusOK},
		{http.MethodGet, "/v1/todos", "", true, http.StatusOK},
		{http.MethodGet, "/v1/todos/t1/", "", true, http.StatusOK},
		{http.MethodGet, "/v1/todos/t2/", "", true, http.StatusOK},
		{http.MethodPost, "/v1/todos", `{}`, true, http.StatusCreated},
		{http.MethodPatch, "/v1/todos/t2/", `{}`, true, http.StatusOK},
		{http.MethodDelete, "/v1/todos/t2/", "", true, http.StatusNoContent},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.auth {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		router.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s %s: got %d", tc.method, tc.path, rec.Code)
		}
	}

	if requests == 0 || len(cache.setKeys) == 0 || len(cache.delKeys) == 0 {
		t.Fatalf("expected upstream/cache activity: requests=%d set=%d del=%d", requests, len(cache.setKeys), len(cache.delKeys))
	}
}

func TestBuildRouterErrors(t *testing.T) {
	token, _ := signToken("secret", user{ID: "u1", Email: "a@example.com"})

	for _, tc := range []struct {
		name   string
		cache  *fakeCache
		client fakeDoer
		method string
		path   string
		auth   string
		status int
	}{
		{"health fail", &fakeCache{pingErr: errors.New("down")}, fakeDoer{}, http.MethodGet, "/healthz", "", http.StatusServiceUnavailable},
		{"missing auth", &fakeCache{}, fakeDoer{}, http.MethodGet, "/v1/todos", "", http.StatusUnauthorized},
		{"signup forward error", &fakeCache{}, fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader(`{"error":"すでに登録されています"}`))}, nil
		}}, http.MethodPost, "/v1/auth/signup", "", http.StatusConflict},
		{"login forward error", &fakeCache{}, fakeDoer{doFn: func(*http.Request) (*http.Response, error) { return nil, errors.New("boom") }}, http.MethodPost, "/v1/auth/login", "", http.StatusBadGateway},
		{"todos upstream error", &fakeCache{}, fakeDoer{doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"bad"}`))}, nil
		}}, http.MethodGet, "/v1/todos", "Bearer " + token, http.StatusBadRequest},
		{"todo delete upstream error", &fakeCache{}, fakeDoer{doFn: func(*http.Request) (*http.Response, error) { return nil, errors.New("boom") }}, http.MethodDelete, "/v1/todos/t1/", "Bearer " + token, http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			buildRouter(config{UserServiceURL: "http://user", TodoServiceURL: "http://todo", AppSessionSecret: "secret", CacheTTL: time.Second}, tc.cache, tc.client).ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("unexpected status: %d", rec.Code)
			}
		})
	}
}

func TestRedisAdapterAndMain(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 10 * time.Millisecond, ReadTimeout: 10 * time.Millisecond, WriteTimeout: 10 * time.Millisecond})
	defer client.Close()
	adapter := &redisClientAdapter{client: client}
	if err := adapter.Ping(context.Background()); err == nil {
		t.Fatal("expected ping error")
	}
	if _, err := adapter.Get(context.Background(), "k"); err == nil {
		t.Fatal("expected get error")
	}
	if err := adapter.Set(context.Background(), "k", []byte("v"), time.Second); err == nil {
		t.Fatal("expected set error")
	}
	if err := adapter.Del(context.Background(), "k"); err == nil {
		t.Fatal("expected del error")
	}

	if os.Getenv("GO_WANT_BFF_MAIN") == "1" {
		_ = os.Unsetenv("USER_SERVICE_URL")
		_ = os.Unsetenv("TODO_SERVICE_URL")
		_ = os.Unsetenv("APP_SESSION_SECRET")
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRedisAdapterAndMain")
	cmd.Env = append(os.Environ(), "GO_WANT_BFF_MAIN=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
}
