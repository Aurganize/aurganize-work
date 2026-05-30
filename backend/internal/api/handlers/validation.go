package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// parseValidationError converts the validator's aggregated error into a
// map of field -> human-readable reason, suitable for the API's "fields"
// response object.
func parseValidationError(err error) map[string]string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		// Body didn't match the struct shape at all (e.g., not valid JSON,
		// type mismatch). Return a generic message under "_body".
		return map[string]string{"_body": err.Error()}
	}

	out := make(map[string]string, len(ve))
	for _, fieldError := range ve {
		field := strings.ToLower(fieldError.Field())
		switch fieldError.Tag() {
		case "required":
			out[field] = "field is required"
		case "email":
			out[field] = "must be a valid email"
		case "max":
			out[field] = fmt.Sprintf("must be at most %s characters", fieldError.Param())
		case "min":
			out[field] = fmt.Sprintf("must be at least %s characters long", fieldError.Param())
		case "oneof":
			out[field] = fmt.Sprintf("must be one of : %s", fieldError.Param())
		default:
			out[field] = fmt.Sprintf("failed %s validation", fieldError.Tag())
		}
	}

	return out
}
