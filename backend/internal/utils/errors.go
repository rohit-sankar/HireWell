package utils

import "net/http"

// AppError is a typed, HTTP-status-aware error that services return and the
// centralized error-handling middleware converts into the standard envelope.
// This is how we "differentiate validation / auth / not found / conflict /
// db / AI / file / internal" without every controller re-deriving it.
type AppError struct {
	Status  int
	Code    string
	Message string
	Details interface{}
	Err     error // wrapped internal error, never sent to the client
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func NewValidationError(message string, details interface{}) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: CodeValidation, Message: message, Details: details}
}

func NewUnauthorized(message string) *AppError {
	if message == "" {
		message = "authentication required"
	}
	return &AppError{Status: http.StatusUnauthorized, Code: CodeUnauthorized, Message: message}
}

func NewForbidden(message string) *AppError {
	if message == "" {
		message = "you do not have access to this resource"
	}
	return &AppError{Status: http.StatusForbidden, Code: CodeForbidden, Message: message}
}

func NewNotFound(resource string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: CodeNotFound, Message: resource + " not found"}
}

func NewConflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: CodeConflict, Message: message}
}

func NewBadRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: CodeBadRequest, Message: message}
}

func NewDatabaseError(err error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: CodeDatabaseError, Message: "a database error occurred", Err: err}
}

func NewAIError(message string, err error) *AppError {
	return &AppError{Status: http.StatusBadGateway, Code: CodeAIError, Message: message, Err: err}
}

func NewFileError(message string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: CodeFileError, Message: message}
}

func NewInternalError(err error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: CodeInternal, Message: "an internal error occurred", Err: err}
}
