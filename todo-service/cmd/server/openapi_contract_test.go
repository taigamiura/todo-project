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
	"time"

	"gopkg.in/yaml.v3"
)

type openAPISpec struct {
	Paths map[string]openAPIPathItem `yaml:"paths"`
}

type openAPIPathItem struct {
	Get    openAPIOperation `yaml:"get"`
	Post   openAPIOperation `yaml:"post"`
	Patch  openAPIOperation `yaml:"patch"`
	Delete openAPIOperation `yaml:"delete"`
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

func loadTodoServiceSpec(t *testing.T) openAPISpec {
	t.Helper()
	path := filepath.Join("..", "..", "..", "openapi", "todo-service.yaml")
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
	pathItem, ok := spec.Paths[path]
	if !ok {
		t.Fatalf("missing path %s", path)
	}

	var operation openAPIOperation
	switch method {
	case "get":
		operation = pathItem.Get
	case "post":
		operation = pathItem.Post
	case "patch":
		operation = pathItem.Patch
	case "delete":
		operation = pathItem.Delete
	default:
		t.Fatalf("unsupported method %s", method)
	}

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

func TestOpenAPIExamplesMatchTodoServiceRoutes(t *testing.T) {
	spec := loadTodoServiceSpec(t)
	item := todo{
		ID:          "todo-1",
		Title:       "Buy milk",
		Description: "2 liters",
		Completed:   false,
		CreatedAt:   time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC),
	}
	router := buildRouter(&fakeTodoStore{
		listFn:   func(context.Context, string) ([]todo, error) { return []todo{item}, nil },
		createFn: func(context.Context, string, todoInput) (todo, error) { return item, nil },
		getFn:    func(context.Context, string, string) (todo, error) { return item, nil },
		updateFn: func(context.Context, string, string, todoInput) (todo, error) { return item, nil },
		deleteFn: func(context.Context, string, string) (bool, error) { return true, nil },
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

	t.Run("list example", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/internal/todos/", nil)
		req.Header.Set("X-User-ID", "user-1")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/todos/", "get", "200"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("create example", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/internal/todos/", strings.NewReader(`{"title":"Buy milk","description":"2 liters","completed":false}`))
		req.Header.Set("X-User-ID", "user-1")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/todos/", "post", "201"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("detail example", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/internal/todos/todo-1/", nil)
		req.Header.Set("X-User-ID", "user-1")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/todos/{id}/", "get", "200"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("update example", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/internal/todos/todo-1/", strings.NewReader(`{"title":"Buy milk","description":"2 liters","completed":false}`))
		req.Header.Set("X-User-ID", "user-1")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/todos/{id}/", "patch", "200"); !deepEqualJSON(actual, expected) {
			t.Fatalf("unexpected body: %#v != %#v", actual, expected)
		}
	})

	t.Run("not found example", func(t *testing.T) {
		notFoundRouter := buildRouter(&fakeTodoStore{getFn: func(context.Context, string, string) (todo, error) { return todo{}, errNotFound }})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/internal/todos/todo-1/", nil)
		req.Header.Set("X-User-ID", "user-1")
		notFoundRouter.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if actual, expected := decodeBody(t, rec), requireExample(t, spec, "/internal/todos/{id}/", "get", "404"); !deepEqualJSON(actual, expected) {
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
