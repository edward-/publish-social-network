// Package domain defines the core entities and interfaces for the social media publisher.
package domain

import "errors"

// Domain-level error definitions.
var (
	// ErrPostValidationFailed indicates that a Post failed validation.
	ErrPostValidationFailed = errors.New("post validation failed")

	// ErrMediaNotFound indicates the specified media file could not be found.
	ErrMediaNotFound = errors.New("media file not found")

	// ErrUnsupportedMediaType indicates the media type is not supported.
	ErrUnsupportedMediaType = errors.New("unsupported media type")

	// ErrPlatformNotSupported indicates a platform is not supported.
	ErrPlatformNotSupported = errors.New("platform not supported")

	// ErrAuthentication indicates an authentication failure.
	ErrAuthentication = errors.New("authentication failed")

	// ErrAuthorization indicates the user is not authorized for this action.
	ErrAuthorization = errors.New("authorization failed")

	// ErrPublishFailed indicates the publish operation failed.
	ErrPublishFailed = errors.New("publish operation failed")

	// ErrTimeout indicates the operation timed out.
	ErrTimeout = errors.New("operation timed out")

	// ErrConfigMissing indicates a required configuration value is missing.
	ErrConfigMissing = errors.New("required configuration value missing")
)

// ValidationError wraps multiple validation errors.
type ValidationError struct {
	Errors []error
}

func (e *ValidationError) Error() string {
	return "validation errors: " + joinErrors(e.Errors)
}

func joinErrors(errs []error) string {
	result := ""
	for i, err := range errs {
		if i > 0 {
			result += ", "
		}
		result += err.Error()
	}
	return result
}

// APIError represents an error from a platform's API.
type APIError struct {
	Platform   Platform
	Code       int
	Message    string
	Original   error
}

func (e *APIError) Error() string {
	return e.Platform.String() + " API error (" + string(rune(e.Code)) + "): " + e.Message
}

func (e *APIError) Unwrap() error {
	return e.Original
}
