package main

import "net/http"

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
	errorCodeBadRequest             = "BAD_REQUEST"
	errorCodeInvalidRequestBody     = "INVALID_REQUEST_BODY"
	errorCodeValidationFailed       = "VALIDATION_FAILED"
	errorCodeDatabaseUnavailable    = "DATABASE_UNAVAILABLE"
	errorCodeUserEmailConflict      = "USER_EMAIL_CONFLICT"
	errorCodeUserCreateFailed       = "USER_CREATE_FAILED"
	errorCodeAuthInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	errorCodeAuthenticationFailed   = "AUTHENTICATION_FAILED"
	errorCodeInternalServerError    = "INTERNAL_SERVER_ERROR"
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
	code := errorCodeInternalServerError
	switch status {
	case http.StatusBadRequest:
		code = errorCodeBadRequest
	case http.StatusServiceUnavailable:
		code = errorCodeDatabaseUnavailable
	}
	writeErrorWithCode(w, status, code, message)
}
