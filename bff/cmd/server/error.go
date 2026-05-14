package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (err *apiError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

type errorResponse struct {
	Error apiError `json:"error"`
}

const (
	errorCodeBadRequest              = "BAD_REQUEST"
	errorCodeUnauthorized            = "UNAUTHORIZED"
	errorCodeNotFound                = "NOT_FOUND"
	errorCodeConflict                = "CONFLICT"
	errorCodeInternalServerError     = "INTERNAL_SERVER_ERROR"
	errorCodeBadGateway              = "BAD_GATEWAY"
	errorCodeServiceUnavailable      = "SERVICE_UNAVAILABLE"
	errorCodeRedisUnavailable        = "REDIS_UNAVAILABLE"
	errorCodeSessionCreateFailed     = "SESSION_CREATE_FAILED"
	errorCodeAuthRequired            = "AUTH_REQUIRED"
	errorCodeInvalidSessionToken     = "INVALID_SESSION_TOKEN"
	errorCodeUserServiceUnavailable  = "USER_SERVICE_UNAVAILABLE"
	errorCodeTodoServiceUnavailable  = "TODO_SERVICE_UNAVAILABLE"
	errorCodeUpstreamResponseInvalid = "UPSTREAM_RESPONSE_INVALID"
	errorCodeUpstreamRequestFailed   = "UPSTREAM_REQUEST_FAILED"
)

func newAPIError(status int, code string, message string) *apiError {
	return &apiError{Code: code, Message: message, Status: status}
}

func writeAPIError(w http.ResponseWriter, err *apiError) {
	writeJSON(w, err.Status, errorResponse{Error: *err})
}

func writeErrorWithCode(w http.ResponseWriter, status int, code string, message string) {
	writeAPIError(w, newAPIError(status, code, message))
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeErrorWithCode(w, status, defaultErrorCode(status), message)
}

func decodeForwardError(payload []byte, fallbackStatus int) error {
	var structured struct {
		Error *apiError `json:"error"`
	}
	if err := json.Unmarshal(payload, &structured); err == nil && structured.Error != nil {
		if strings.TrimSpace(structured.Error.Message) != "" {
			if structured.Error.Status == 0 {
				structured.Error.Status = fallbackStatus
			}
			if strings.TrimSpace(structured.Error.Code) == "" {
				structured.Error.Code = defaultErrorCode(fallbackStatus)
			}
			return structured.Error
		}
	}

	var legacy struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(payload, &legacy); err == nil {
		if message := strings.TrimSpace(legacy.Error); message != "" {
			return newAPIError(fallbackStatus, defaultErrorCode(fallbackStatus), message)
		}
	}

	return newAPIError(fallbackStatus, defaultErrorCode(fallbackStatus), fmt.Sprintf("upstream request failed with status %d", fallbackStatus))
}

func writeForwardError(w http.ResponseWriter, err error) {
	var appErr *apiError
	if errors.As(err, &appErr) {
		writeAPIError(w, appErr)
		return
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "すでに登録"):
		writeErrorWithCode(w, http.StatusConflict, errorCodeConflict, message)
	case strings.Contains(message, "正しくありません"):
		writeErrorWithCode(w, http.StatusUnauthorized, errorCodeUnauthorized, message)
	case strings.Contains(message, "必須"), strings.Contains(message, "文字"):
		writeErrorWithCode(w, http.StatusBadRequest, errorCodeBadRequest, message)
	default:
		writeErrorWithCode(w, http.StatusBadGateway, errorCodeBadGateway, message)
	}
}

func defaultErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return errorCodeBadRequest
	case http.StatusUnauthorized:
		return errorCodeUnauthorized
	case http.StatusNotFound:
		return errorCodeNotFound
	case http.StatusConflict:
		return errorCodeConflict
	case http.StatusServiceUnavailable:
		return errorCodeServiceUnavailable
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return errorCodeBadGateway
	default:
		return errorCodeInternalServerError
	}
}
