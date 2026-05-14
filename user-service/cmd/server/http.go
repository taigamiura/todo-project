package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func buildRouter(store userStore) http.Handler {
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

	router.Post("/internal/users/signup", func(w http.ResponseWriter, r *http.Request) {
		var input authInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeErrorWithCode(w, http.StatusBadRequest, errorCodeInvalidRequestBody, "invalid request body")
			return
		}

		created, err := store.CreateUser(r.Context(), input)
		if err != nil {
			switch {
			case errors.Is(err, errConflict):
				writeErrorWithCode(w, http.StatusConflict, errorCodeUserEmailConflict, "このメールアドレスはすでに登録されています。")
			case errors.Is(err, errValidation):
				writeErrorWithCode(w, http.StatusBadRequest, errorCodeValidationFailed, err.Error())
			default:
				writeErrorWithCode(w, http.StatusInternalServerError, errorCodeUserCreateFailed, "user creation failed")
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]user{"user": created})
	})

	router.Post("/internal/users/login", func(w http.ResponseWriter, r *http.Request) {
		var input authInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeErrorWithCode(w, http.StatusBadRequest, errorCodeInvalidRequestBody, "invalid request body")
			return
		}

		found, err := store.AuthenticateUser(r.Context(), input)
		if err != nil {
			switch {
			case errors.Is(err, errUnauthorized):
				writeErrorWithCode(w, http.StatusUnauthorized, errorCodeAuthInvalidCredentials, "メールアドレスまたはパスワードが正しくありません。")
			case errors.Is(err, errValidation):
				writeErrorWithCode(w, http.StatusBadRequest, errorCodeValidationFailed, err.Error())
			default:
				writeErrorWithCode(w, http.StatusInternalServerError, errorCodeAuthenticationFailed, "authentication failed")
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]user{"user": found})
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
