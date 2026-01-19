// Package errors provides custom error types for user-friendly error messaging.
//
// This package distinguishes between user-facing errors and technical errors,
// allowing the CLI to display clean messages while preserving technical details
// for debugging with verbose flags.
package errors

import (
	"errors"
	"fmt"
)

// UserError represents an error with a user-friendly message.
//
// UserError separates user-facing messages from technical implementation details,
// making CLI output cleaner while preserving debugging information for verbose mode.
type UserError struct {
	// Message is the user-friendly error message displayed to users.
	Message string

	// Err is the underlying technical error, preserved for debugging
	// but hidden from normal output.
	Err error

	// Hint provides actionable guidance to help users resolve the issue.
	Hint string
}

// Error implements the error interface and returns the user-friendly message.
//
// If a hint is set, it appends the hint to the message on a new line.
func (e *UserError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s\n%s", e.Message, e.Hint)
	}
	return e.Message
}

// Unwrap returns the underlying technical error for error chain inspection.
func (e *UserError) Unwrap() error {
	return e.Err
}

// IsUserError checks whether an error chain contains a UserError.
//
// It uses errors.As to walk the error chain and returns the first UserError found.
// The second return value indicates whether a UserError was found.
func IsUserError(err error) (*UserError, bool) {
	var userErr *UserError
	if errors.As(err, &userErr) {
		return userErr, true
	}
	return nil, false
}

// NewUserError creates a user-facing error with a message.
//
// Use this for simple errors that don't need hints or wrapped technical errors.
func NewUserError(message string) *UserError {
	return &UserError{Message: message}
}

// NewUserErrorWithHint creates a user-facing error with a message and actionable hint.
//
// The hint should provide specific instructions to help users resolve the issue,
// such as command examples or documentation links.
func NewUserErrorWithHint(message, hint string) *UserError {
	return &UserError{Message: message, Hint: hint}
}

// WrapUserError wraps a technical error with a user-friendly message.
//
// The technical error is preserved for debugging with verbose flags but hidden
// from normal output.
func WrapUserError(message string, err error) *UserError {
	return &UserError{Message: message, Err: err}
}

// WrapUserErrorWithHint wraps a technical error with a user-friendly message and hint.
//
// This combines a clean user message, actionable guidance, and technical details
// for debugging.
func WrapUserErrorWithHint(message, hint string, err error) *UserError {
	return &UserError{Message: message, Hint: hint, Err: err}
}
