// Package utils holds small cross-cutting helpers: the API response
// envelope, error codes, and password/slug helpers used by multiple
// domains.
package utils

import "github.com/gin-gonic/gin"

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type ErrorResponse struct {
	Success bool      `json:"success"`
	Error   ErrorBody `json:"error"`
}

func Success(c *gin.Context, status int, data interface{}, message string) {
	c.JSON(status, SuccessResponse{Success: true, Data: data, Message: message})
}

func SuccessWithMeta(c *gin.Context, status int, data interface{}, message string, meta interface{}) {
	c.JSON(status, SuccessResponse{Success: true, Data: data, Message: message, Meta: meta})
}

func Error(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// Common, reusable error codes. Keeping these centralized prevents the
// classic problem of every controller inventing its own string.
const (
	CodeValidation    = "VALIDATION_ERROR"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
	CodeNotFound      = "NOT_FOUND"
	CodeConflict      = "CONFLICT"
	CodeInternal      = "INTERNAL_ERROR"
	CodeDatabaseError = "DATABASE_ERROR"
	CodeAIError       = "AI_PROVIDER_ERROR"
	CodeFileError     = "FILE_ERROR"
	CodeBadRequest    = "BAD_REQUEST"
	CodeRateLimited   = "RATE_LIMITED"
)
