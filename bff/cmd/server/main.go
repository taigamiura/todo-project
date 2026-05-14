package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	Port             string
	UserServiceURL   string
	TodoServiceURL   string
	AppSessionSecret string
	CacheTTL         time.Duration
	RedisAddr        string
}

type user struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type authResponse struct {
	User user `json:"user"`
}

type sessionResponse struct {
	AccessToken string `json:"accessToken"`
	User        user   `json:"user"`
}

type todo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type todoListResponse struct {
	Todos []todo `json:"todos"`
}

type todoResponse struct {
	Todo todo `json:"todo"`
}

type contextKey string

const userContextKey contextKey = "user"

type redisCache interface {
	Ping(context.Context) error
	Get(context.Context, string) (string, error)
	Set(context.Context, string, []byte, time.Duration) error
	Del(context.Context, ...string) error
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func main() {
	ctx, stop := setupRuntimeContext()
	defer stop()

	telemetryShutdown, err := setupTelemetry(ctx, serviceName)
	if err != nil {
		appLogger.Error("telemetry_init_failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = telemetryShutdown(shutdownCtx)
	}()

	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		appLogger.Error("config_load_failed", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		PoolTimeout:  time.Second,
	})
	if err := redisotel.InstrumentTracing(redisClient, redisotel.WithDBStatement(false)); err != nil {
		appLogger.Error("redis_tracing_init_failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 4 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: otelhttp.NewTransport(transport)}
	router := buildRouter(cfg, &redisClientAdapter{client: redisClient}, httpClient)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	appLogger.Info("server_start", "addr", server.Addr)
	if err := runHTTPServer(ctx, server, appLogger); err != nil {
		appLogger.Error("server_failed", "error", err)
		os.Exit(1)
	}
}

type redisClientAdapter struct {
	client *redis.Client
}

func (adapter *redisClientAdapter) Ping(ctx context.Context) error {
	ctx, span := otel.Tracer(serviceName).Start(ctx, "redis PING")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "PING"),
	)
	return adapter.client.Ping(ctx).Err()
}

func (adapter *redisClientAdapter) Get(ctx context.Context, key string) (string, error) {
	ctx, span := otel.Tracer(serviceName).Start(ctx, "redis GET")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "GET"),
		attribute.String("db.redis.key", key),
	)
	return adapter.client.Get(ctx, key).Result()
}

func (adapter *redisClientAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, span := otel.Tracer(serviceName).Start(ctx, "redis SET")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "SET"),
		attribute.String("db.redis.key", key),
		attribute.Int("db.redis.value_size", len(value)),
		attribute.String("db.redis.ttl", ttl.String()),
	)
	return adapter.client.Set(ctx, key, value, ttl).Err()
}

func (adapter *redisClientAdapter) Del(ctx context.Context, keys ...string) error {
	ctx, span := otel.Tracer(serviceName).Start(ctx, "redis DEL")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "DEL"),
		attribute.Int("db.redis.key_count", len(keys)),
	)
	return adapter.client.Del(ctx, keys...).Err()
}

func buildRouter(cfg config, cache redisCache, httpClient httpDoer) http.Handler {
	router := chi.NewRouter()
	attachRuntimeMiddleware(router)
	router.Use(timeoutMiddleware(8 * time.Second))
	router.Method(http.MethodGet, "/metrics", promhttp.Handler())

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := cache.Ping(r.Context()); err != nil {
			writeErrorWithCode(w, http.StatusServiceUnavailable, errorCodeRedisUnavailable, "redis is unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Post("/v1/auth/signup", func(w http.ResponseWriter, r *http.Request) {
		userPayload, err := forwardAuth(r.Context(), httpClient, cfg.UserServiceURL+"/internal/users/signup", r.Body)
		if err != nil {
			writeForwardError(w, err)
			return
		}

		token, err := signTokenWithContext(r.Context(), cfg.AppSessionSecret, userPayload)
		if err != nil {
			writeErrorWithCode(w, http.StatusInternalServerError, errorCodeSessionCreateFailed, "failed to create session")
			return
		}

		writeJSON(w, http.StatusCreated, sessionResponse{AccessToken: token, User: userPayload})
	})

	router.Post("/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		userPayload, err := forwardAuth(r.Context(), httpClient, cfg.UserServiceURL+"/internal/users/login", r.Body)
		if err != nil {
			writeForwardError(w, err)
			return
		}

		token, err := signTokenWithContext(r.Context(), cfg.AppSessionSecret, userPayload)
		if err != nil {
			writeErrorWithCode(w, http.StatusInternalServerError, errorCodeSessionCreateFailed, "failed to create session")
			return
		}

		writeJSON(w, http.StatusOK, sessionResponse{AccessToken: token, User: userPayload})
	})

	router.Group(func(r chi.Router) {
		r.Use(authMiddleware(cfg.AppSessionSecret))

		r.Get("/v1/todos", func(w http.ResponseWriter, r *http.Request) {
			currentUser := r.Context().Value(userContextKey).(user)
			cacheKey := fmt.Sprintf("todos:%s", currentUser.ID)
			span := trace.SpanFromContext(r.Context())

			if cached, err := cache.Get(r.Context(), cacheKey); err == nil {
				span.SetAttributes(
					attribute.Bool("cache.hit", true),
					attribute.String("cache.keyspace", "todos"),
				)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(cached))
				return
			}

			span.SetAttributes(
				attribute.Bool("cache.hit", false),
				attribute.String("cache.keyspace", "todos"),
			)

			body, status, err := proxyTodoRequest(r.Context(), httpClient, cfg.TodoServiceURL+"/internal/todos/", currentUser.ID, http.MethodGet, nil)
			if err != nil {
				writeForwardError(w, err)
				return
			}

			_ = cache.Set(r.Context(), cacheKey, body, cfg.CacheTTL)
			writeRawJSON(w, status, body)
		})

		r.Post("/v1/todos", func(w http.ResponseWriter, r *http.Request) {
			currentUser := r.Context().Value(userContextKey).(user)
			body, status, err := proxyTodoRequest(r.Context(), httpClient, cfg.TodoServiceURL+"/internal/todos/", currentUser.ID, http.MethodPost, r.Body)
			if err != nil {
				writeForwardError(w, err)
				return
			}

			invalidateTodoCache(r.Context(), cache, currentUser.ID, "")
			writeRawJSON(w, status, body)
		})

		r.Route("/v1/todos/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				currentUser := r.Context().Value(userContextKey).(user)
				todoID := chi.URLParam(r, "id")
				cacheKey := fmt.Sprintf("todo:%s:%s", currentUser.ID, todoID)
				span := trace.SpanFromContext(r.Context())

				if cached, err := cache.Get(r.Context(), cacheKey); err == nil {
					span.SetAttributes(
						attribute.Bool("cache.hit", true),
						attribute.String("cache.keyspace", "todo"),
					)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(cached))
					return
				}

				span.SetAttributes(
					attribute.Bool("cache.hit", false),
					attribute.String("cache.keyspace", "todo"),
				)

				body, status, err := proxyTodoRequest(r.Context(), httpClient, cfg.TodoServiceURL+"/internal/todos/"+todoID+"/", currentUser.ID, http.MethodGet, nil)
				if err != nil {
					writeForwardError(w, err)
					return
				}

				_ = cache.Set(r.Context(), cacheKey, body, cfg.CacheTTL)
				writeRawJSON(w, status, body)
			})

			r.Patch("/", func(w http.ResponseWriter, r *http.Request) {
				currentUser := r.Context().Value(userContextKey).(user)
				todoID := chi.URLParam(r, "id")
				body, status, err := proxyTodoRequest(r.Context(), httpClient, cfg.TodoServiceURL+"/internal/todos/"+todoID+"/", currentUser.ID, http.MethodPatch, r.Body)
				if err != nil {
					writeForwardError(w, err)
					return
				}

				invalidateTodoCache(r.Context(), cache, currentUser.ID, todoID)
				writeRawJSON(w, status, body)
			})

			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				currentUser := r.Context().Value(userContextKey).(user)
				todoID := chi.URLParam(r, "id")
				body, status, err := proxyTodoRequest(r.Context(), httpClient, cfg.TodoServiceURL+"/internal/todos/"+todoID+"/", currentUser.ID, http.MethodDelete, nil)
				if err != nil {
					writeForwardError(w, err)
					return
				}

				invalidateTodoCache(r.Context(), cache, currentUser.ID, todoID)
				if status == http.StatusNoContent {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				writeRawJSON(w, status, body)
			})
		})
	})

	return router
}

func loadConfig(getenv func(string) string) (config, error) {
	port := getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cacheTTL, err := time.ParseDuration(defaultString(getenv("CACHE_TTL"), "30s"))
	if err != nil {
		cacheTTL = 30 * time.Second
	}

	cfg := config{
		Port:             port,
		UserServiceURL:   getenv("USER_SERVICE_URL"),
		TodoServiceURL:   getenv("TODO_SERVICE_URL"),
		AppSessionSecret: getenv("APP_SESSION_SECRET"),
		CacheTTL:         cacheTTL,
		RedisAddr:        defaultString(getenv("REDIS_ADDR"), "redis:6379"),
	}

	if cfg.UserServiceURL == "" || cfg.TodoServiceURL == "" || cfg.AppSessionSecret == "" {
		return config{}, fmt.Errorf("USER_SERVICE_URL, TODO_SERVICE_URL and APP_SESSION_SECRET are required")
	}

	return cfg, nil
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func forwardAuth(ctx context.Context, client httpDoer, url string, body io.Reader) (user, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return user{}, newAPIError(http.StatusInternalServerError, errorCodeUpstreamRequestFailed, "failed to prepare upstream request")
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return user{}, newAPIError(http.StatusBadGateway, errorCodeUserServiceUnavailable, "user service is unavailable")
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return user{}, newAPIError(http.StatusBadGateway, errorCodeUpstreamResponseInvalid, "failed to read upstream response")
	}

	if response.StatusCode >= http.StatusBadRequest {
		return user{}, decodeForwardError(payload, response.StatusCode)
	}

	var decoded authResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return user{}, newAPIError(http.StatusBadGateway, errorCodeUpstreamResponseInvalid, "failed to decode upstream response")
	}

	return decoded.User, nil
}

func signToken(secret string, currentUser user) (string, error) {
	return signTokenWithContext(context.Background(), secret, currentUser)
}

func signTokenWithContext(ctx context.Context, secret string, currentUser user) (string, error) {
	_, span := otel.Tracer(serviceName).Start(ctx, "session.issue")
	defer span.End()
	span.SetAttributes(attribute.Bool("user.authenticated", currentUser.ID != ""))

	claims := jwt.MapClaims{
		"sub":   currentUser.ID,
		"name":  currentUser.Name,
		"email": currentUser.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(12 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func authMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(header, "Bearer ") {
				writeErrorWithCode(w, http.StatusUnauthorized, errorCodeAuthRequired, "認証が必要です。")
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeErrorWithCode(w, http.StatusUnauthorized, errorCodeInvalidSessionToken, "認証が必要です。")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeErrorWithCode(w, http.StatusUnauthorized, errorCodeInvalidSessionToken, "認証が必要です。")
				return
			}

			currentUser := user{
				ID:    fmt.Sprint(claims["sub"]),
				Name:  fmt.Sprint(claims["name"]),
				Email: fmt.Sprint(claims["email"]),
			}
			if currentUser.ID == "" || currentUser.Email == "" {
				writeErrorWithCode(w, http.StatusUnauthorized, errorCodeInvalidSessionToken, "認証が必要です。")
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, currentUser)))
		})
	}
}

func proxyTodoRequest(ctx context.Context, client httpDoer, url string, userID string, method string, body io.Reader) ([]byte, int, error) {
	_, span := otel.Tracer(serviceName).Start(ctx, "todo.upstream_request")
	defer span.End()
	span.SetAttributes(
		attribute.String("upstream.service", "todo-service"),
		attribute.String("http.request.method", method),
	)

	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, http.StatusInternalServerError, newAPIError(http.StatusInternalServerError, errorCodeUpstreamRequestFailed, "failed to prepare upstream request")
	}
	request.Header.Set("X-User-ID", userID)
	if method == http.MethodPost || method == http.MethodPatch {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, http.StatusBadGateway, newAPIError(http.StatusBadGateway, errorCodeTodoServiceUnavailable, "todo service is unavailable")
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, http.StatusBadGateway, newAPIError(http.StatusBadGateway, errorCodeUpstreamResponseInvalid, "failed to read upstream response")
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, response.StatusCode, decodeForwardError(payload, response.StatusCode)
	}
	span.SetAttributes(attribute.Int("http.response.status_code", response.StatusCode))

	return payload, response.StatusCode, nil
}

func invalidateTodoCache(ctx context.Context, client redisCache, userID string, todoID string) {
	_, span := otel.Tracer(serviceName).Start(ctx, "cache.invalidate")
	defer span.End()
	span.SetAttributes(attribute.String("cache.keyspace", "todos"))

	keys := []string{fmt.Sprintf("todos:%s", userID)}
	if todoID != "" {
		keys = append(keys, fmt.Sprintf("todo:%s:%s", userID, todoID))
		span.SetAttributes(attribute.String("cache.keyspace", "todos,todo"))
	}
	span.SetAttributes(attribute.Int("cache.key_count", len(keys)))
	_ = client.Del(ctx, keys...)
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

func writeRawJSON(w http.ResponseWriter, status int, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.Copy(w, bytes.NewReader(payload))
}
