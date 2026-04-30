package data

import (
	"context"
	"errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// ErrorSeverity distinguishes retriable (Warning) from hard (Error) failures.
type ErrorSeverity int

const (
	ErrorSeverityWarning ErrorSeverity = iota // transient / retriable
	ErrorSeverityError                        // hard / requires operator action
)

// SeverityOf classifies err using rc's classifier methods.
// Classification order: Unauthorized → Forbidden → NotFound → Timeout → Generic.
func SeverityOf(err error, rc ResourceClient) ErrorSeverity {
	if err == nil {
		return ErrorSeverityWarning
	}
	if rc.IsUnauthorized(err) || rc.IsForbidden(err) || rc.IsNotFound(err) {
		return ErrorSeverityError
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorSeverityWarning
	}
	return ErrorSeverityWarning
}

// SeverityOfClassified classifies err using k8s typed errors directly — no
// ResourceClient needed. Use this in components that don't carry an rc field.
func SeverityOfClassified(err error) ErrorSeverity {
	if err == nil {
		return ErrorSeverityWarning
	}
	if k8serrors.IsUnauthorized(err) || k8serrors.IsForbidden(err) || k8serrors.IsNotFound(err) {
		return ErrorSeverityError
	}
	return ErrorSeverityWarning
}
