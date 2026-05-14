package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type openAPISpec struct {
	Paths map[string]map[string]openAPIOperation `yaml:"paths"`
}

type openAPIOperation struct {
	Responses map[string]openAPIResponse `yaml:"responses"`
}

type openAPIResponse struct {
	Content map[string]openAPIMediaType `yaml:"content"`
}

type openAPIMediaType struct {
	Example any `yaml:"example"`
}

func loadUserServiceSpec(t *testing.T) openAPISpec {
	t.Helper()
	path := filepath.Join("..", "..", "..", "openapi", "user-service.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}

	var spec openAPISpec
	if err := yaml.Unmarshal(content, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}

	return spec
}

func requireExample(t *testing.T, spec openAPISpec, path string, method string, status string) any {
	t.Helper()
	operation := spec.Paths[path][method]
	response, ok := operation.Responses[status]
	if !ok {
		t.Fatalf("missing response %s %s %s", method, path, status)
	}
	mediaType, ok := response.Content["application/json"]
	if !ok {
		t.Fatalf("missing application/json response for %s %s %s", method, path, status)
	}
	if mediaType.Example == nil {
		t.Fatalf("missing example for %s %s %s", method, path, status)
	}
	return mediaType.Example
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) any {
	t.Helper()
	var actual any
	if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return actual
}

func TestOpenAPIExamplesMatchUserServiceRoutes(t *testing.T) {
	spec := loadUserServiceSpec(t)
	router := buildRouter(&fakeUserStore{
		createFn: func(_ context.Context, input authInput) (user, error) {
			return user{ID: "user-1", Name: input.Name, Email: input.Email}, nil
		},
		authFn: func(_ context.Context, _ authInput) (user, error) {
			return user{ID: "user-1", Name: "Taro Todo", Email: "taro@example.com"}, nil
		},
	})

	t.Run("health example", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/healthz", "get", "200"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("signup success example", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/internal/users/signup", strings.NewReader(`{"name":"Taro Todo","email":"taro@example.com","password":"password123"}`))
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/users/signup", "post", "201"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("signup conflict example", func(t *testing.T) {
		conflictRouter := buildRouter(&fakeUserStore{createFn: func(context.Context, authInput) (user, error) { return user{}, errConflict }})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/internal/users/signup", strings.NewReader(`{"name":"Taro Todo","email":"taro@example.com","password":"password123"}`))
		conflictRouter.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/users/signup", "post", "409"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("login success example", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/internal/users/login", strings.NewReader(`{"email":"taro@example.com","password":"password123"}`))
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/users/login", "post", "200"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("login unauthorized example", func(t *testing.T) {
		unauthorizedRouter := buildRouter(&fakeUserStore{authFn: func(context.Context, authInput) (user, error) { return user{}, errUnauthorized }})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/internal/users/login", strings.NewReader(`{"email":"taro@example.com","password":"password123"}`))
		unauthorizedRouter.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/users/login", "post", "401"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})
}

func deepEqualJSON(actual any, expected any) bool {
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return false
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return false
	}
	return string(actualJSON) == string(expectedJSON)
}
