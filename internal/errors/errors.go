// Package errors provides custom error types for user-friendly error messaging.
//
// This package distinguishes between user-facing errors and technical errors,
// allowing the CLI to display clean messages while preserving technical details
// for debugging with verbose flags.
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
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

	// Code is an optional machine-readable identifier (e.g., "AUTH_EXPIRED")
	// for AI agents and other programmatic consumers to branch on.
	Code string

	// Retryable indicates whether the failed operation can be safely retried
	// without further user action.
	Retryable bool
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

// Output formats supported by Format.
const (
	FormatHuman = "human"
	FormatJSON  = "json"
	FormatYAML  = "yaml"
)

// envelope is the structured payload emitted in json/yaml mode.
type envelope struct {
	Error envelopeError `json:"error"`
}

type envelopeError struct {
	Code      string `json:"code,omitempty"`
	Message   string `json:"message"`
	Hint      string `json:"hint,omitempty"`
	Retryable bool   `json:"retryable"`
	Details   string `json:"details,omitempty"`
}

// Format writes err to w in the requested output format.
//
// For "human" (or any unknown value), output mirrors the legacy
// "error: <message>\n<hint>" form, with the wrapped technical error appended
// when verbosity is at least 4. For "json" and "yaml", a structured envelope
// is emitted using the UserError fields when available, falling back to
// err.Error() for non-UserError values.
func Format(w io.Writer, err error, format string, verbosity int) {
	if err == nil {
		return
	}

	userErr, isUser := IsUserError(err)

	switch format {
	case FormatJSON, FormatYAML:
		env := envelope{Error: envelopeError{Message: err.Error()}}
		if isUser {
			env.Error.Code = userErr.Code
			env.Error.Message = userErr.Message
			env.Error.Hint = userErr.Hint
			env.Error.Retryable = userErr.Retryable
			if userErr.Err != nil {
				env.Error.Details = userErr.Err.Error()
			}
		}

		var (
			data []byte
			marshalErr error
		)
		if format == FormatJSON {
			data, marshalErr = json.Marshal(env)
		} else {
			data, marshalErr = yaml.Marshal(env)
		}
		if marshalErr != nil {
			fmt.Fprintf(w, "error: %s\n", err.Error())
			return
		}
		w.Write(data)
		if format == FormatJSON {
			fmt.Fprintln(w)
		}

	default:
		if isUser {
			fmt.Fprintf(w, "error: %s\n", userErr.Error())
			if verbosity >= 4 && userErr.Err != nil {
				fmt.Fprintf(w, "\nDetails:\n%v\n", userErr.Err)
			}
			return
		}
		fmt.Fprintf(w, "error: %s\n", err.Error())
	}
}
