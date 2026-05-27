package response

import (
	"errors"
	"log/slog"
	"net/http"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// ErrorBody is the JSON shape every error response uses.
// Keep it small and stable; clients rely on these field names.
type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// RenderError maps any error to an HTTP response. This is the only place
// errors are turned into JSON. Handlers call it; middleware calls it via
// AbortWithError. Centralising means consistent status codes across the API.
func RenderError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Translate pgx no-rows into our NotFound. This lets storage-layer
	// callers return the raw pgx error without wrapping every single call site.
	if errors.Is(err, pgx.ErrNoRows) {
		err = domain.ErrNotFound("resource", err)
	}

	appErr := domain.AsAppError(err)
	status := statusFor(appErr.Code())

	// Log the cause at the appropriate level. Client-induced errors (4xx)
	// are info-level (noisy but expected); 5xx are error-level.
	logger := logger.FromContext(c.Request.Context())
	if status >= 500 {
		logger.Error("request failed",
			slog.String("code", appErr.Code()),
			slog.String("message", appErr.Message()),
			slog.Any("cause", appErr.Cause()))
	} else {
		logger.Info("request rejected",
			slog.String("code", appErr.Code()),
			slog.String("message", appErr.Message()),
			slog.Any("cause", appErr.Cause()))
	}

	c.AbortWithStatusJSON(status, ErrorBody{
		Code:    appErr.Code(),
		Message: appErr.Message(),
	})
}

// RenderValidationError is a specialised variant that includes per-field
// errors in the response. Handlers call this when ShouldBindJSON or
// validator returns a structured error.
func RenderValidationErrors(c *gin.Context, fields map[string]any) {
	body := ErrorBody{
		Code:    "INVALID_INPUT",
		Message: "Request validation failed",
		Fields:  make(map[string]any),
	}

	for k, v := range fields {
		body.Fields[k] = v
	}

	c.AbortWithStatusJSON(http.StatusBadRequest, body)
}

// statusFor maps an AppError code to an HTTP status code.
func statusFor(code string) int {
	switch code {
	case "NOT_FOUND":
		return http.StatusNotFound
	case "INVALID_INPUT":
		return http.StatusBadRequest
	case "UNAUTHENTICATED":
		return http.StatusUnauthorized
	case "FORBIDDEN":
		return http.StatusForbidden
	case "CONFLICT":
		return http.StatusConflict
	case "BUSINESS_RULE_VIOLATION":
		return http.StatusUnprocessableEntity
	case "INTERNAL":
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// OK writes a 200 with the given body.
func OK(c *gin.Context, body any) {
	c.JSON(http.StatusOK, body)
}

// Created writes a 201 with the given body.
func Created(c *gin.Context, body any) {
	c.JSON(http.StatusCreated, body)
}

// NoContent writes a 204 with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
