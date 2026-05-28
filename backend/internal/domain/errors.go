package domain

import (
	"errors"
	"fmt"
)

// AppError is the interface every domain error implements. The HTTP layer
// uses Code() to choose a status; the Message() is sent to the client.
// Internal details stay in Cause() and get logged but not returned.
type AppError interface {
	error
	Code() string
	Message() string
	Cause() error
}

// baseError is the concrete implementation. All factory functions below
// return *baseError under the AppError interface.
type baseError struct {
	code    string
	message string
	cause   error
}

func (e *baseError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s; %v", e.code, e.message, e.cause)
	}

	return fmt.Sprintf("%s: %s", e.code, e.message)
}

func (e *baseError) Code() string    { return e.code }
func (e *baseError) Message() string { return e.message }
func (e *baseError) Cause() error    { return e.cause }
func (e *baseError) Unwrap() error   { return e.cause }

// === Factory functions ===
//
// Each factory creates a specific error class. Use these from services;
// the HTTP layer maps them to status codes in renderError().

// ErrNotFound — HTTP 404. The requested resource does not exist (or RLS
// hides it, which is indistinguishable to the caller — intentional).
func ErrNotFound(resource string, cause error) AppError {
	return &baseError{
		code:    "NOT_FOUND",
		message: fmt.Sprintf("%s not found", resource),
		cause:   cause,
	}
}

// ErrInvalidInput — HTTP 400. Input failed validation (shape, type, range).
// Used for things validator can't express in struct tags.
func ErrInvalidInput(reason string, cause error) AppError {
	return &baseError{
		code:    "INVALID_INPUT",
		message: reason,
		cause:   cause,
	}
}

// ErrUnauthenticated — HTTP 401. The request has no credentials or invalid ones.
func ErrUnauthenticated(reason string, cause error) AppError {
	return &baseError{
		code:    "UNAUTHENTICATED",
		message: reason,
		cause:   cause,
	}
}

// ErrForbidden — HTTP 403. The caller is authenticated but lacks permission.
func ErrForbidden(reason string, cause error) AppError {
	return &baseError{
		code:    "FORBIDDEN",
		message: reason,
		cause:   cause,
	}
}

// ErrConflict — HTTP 409. A unique constraint, optimistic-lock, or business
// rule precludes the operation. E.g., email already in use.
func ErrConflict(reason string, cause error) AppError {
	return &baseError{
		code:    "CONFLICT",
		message: reason,
		cause:   cause,
	}
}

// ErrBusinessRule — HTTP 422. A semantically valid request that violates
// a business rule (e.g., can't advance a project past its current stage).
func ErrBusinessRuleViolation(reason string, cause error) AppError {
	return &baseError{
		code:    "BUSINESS_RULE_VIOLATION",
		message: reason,
		cause:   cause,
	}
}

// ErrInternal — HTTP 500. Something unexpected. Message is opaque; cause
// is logged. Use this when you don't have a more specific error.
func ErrInternal(cause error) AppError {
	return &baseError{
		code:    "INTERNAL",
		message: "An internal error occured",
		cause:   cause,
	}
}

// AsAppError extracts an AppError from any error, or wraps it as ErrInternal
// if it's not one. The HTTP layer uses this to ensure every error has a
// status code mapping.
func AsAppError(err error) AppError {
	if err == nil {
		return nil
	}

	var ae AppError
	if errors.As(err, &ae) {
		return ae
	}

	return ErrInternal(err)
}
