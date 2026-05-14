package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func buildRouter(store todoStore) http.Handler {
	router := chi.NewRouter()
	attachRuntimeMiddleware(router)
	router.Use(timeoutMiddleware(5 * time.Second))
	router.Method(http.MethodGet, "/metrics", promhttp.Handler())

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := store.Ping(r.Context()); err != nil {
			writeErrorWithCode(w, http.StatusServiceUnavailable, errorCodeDatabaseUnavailable, "database is unavailable")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Route("/internal/todos", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
			if userID == "" {
				writeErrorWithCode(w, http.StatusUnauthorized, errorCodeAuthContextMissing, "missing user context")
				return
			}

			todos, err := store.ListTodos(r.Context(), userID)
			if err != nil {
				writeErrorWithCode(w, http.StatusInternalServerError, errorCodeTodoListFailed, "failed to load todos")
				return
			}

			writeJSON(w, http.StatusOK, map[string][]todo{"todos": todos})
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
			if userID == "" {
				writeErrorWithCode(w, http.StatusUnauthorized, errorCodeAuthContextMissing, "missing user context")
				return
			}

			var input todoInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeErrorWithCode(w, http.StatusBadRequest, errorCodeInvalidRequestBody, "invalid request body")
				return
			}
			if err := validateTodoInput(input); err != nil {
				writeErrorWithCode(w, http.StatusBadRequest, errorCodeValidationFailed, err.Error())
				return
			}

			item, err := store.CreateTodo(r.Context(), userID, input)
			if err != nil {
				writeErrorWithCode(w, http.StatusInternalServerError, errorCodeTodoCreateFailed, "failed to create todo")
				return
			}

			writeJSON(w, http.StatusCreated, map[string]todo{"todo": item})
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				item, err := store.GetTodo(r.Context(), chi.URLParam(r, "id"), r.Header.Get("X-User-ID"))
				if err != nil {
					if errors.Is(err, errNotFound) {
						writeErrorWithCode(w, http.StatusNotFound, errorCodeTodoNotFound, "Todo が見つかりません。")
						return
					}
					writeErrorWithCode(w, http.StatusInternalServerError, errorCodeTodoFetchFailed, "failed to load todo")
					return
				}

				writeJSON(w, http.StatusOK, map[string]todo{"todo": item})
			})

			r.Patch("/", func(w http.ResponseWriter, r *http.Request) {
				userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
				if userID == "" {
					writeErrorWithCode(w, http.StatusUnauthorized, errorCodeAuthContextMissing, "missing user context")
					return
				}

				var input todoInput
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					writeErrorWithCode(w, http.StatusBadRequest, errorCodeInvalidRequestBody, "invalid request body")
					return
				}
				if err := validateTodoInput(input); err != nil {
					writeErrorWithCode(w, http.StatusBadRequest, errorCodeValidationFailed, err.Error())
					return
				}

				item, err := store.UpdateTodo(r.Context(), chi.URLParam(r, "id"), userID, input)
				if err != nil {
					if errors.Is(err, errNotFound) {
						writeErrorWithCode(w, http.StatusNotFound, errorCodeTodoNotFound, "Todo が見つかりません。")
						return
					}
					writeErrorWithCode(w, http.StatusInternalServerError, errorCodeTodoUpdateFailed, "failed to update todo")
					return
				}

				writeJSON(w, http.StatusOK, map[string]todo{"todo": item})
			})

			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleted, err := store.DeleteTodo(r.Context(), chi.URLParam(r, "id"), r.Header.Get("X-User-ID"))
				if err != nil {
					writeErrorWithCode(w, http.StatusInternalServerError, errorCodeTodoDeleteFailed, "failed to delete todo")
					return
				}
				if !deleted {
					writeErrorWithCode(w, http.StatusNotFound, errorCodeTodoNotFound, "Todo が見つかりません。")
					return
				}

				w.WriteHeader(http.StatusNoContent)
			})
		})
	})

	return router
}

func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "request timed out")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
