// Package fault defines stable errors shared by v2 components.
package fault

import (
	"errors"
	"fmt"
)

// Kind is a machine-readable failure category.
type Kind string

const (
	KindInvalidInput   Kind = "invalid-input"
	KindUnavailable    Kind = "unavailable"
	KindTimeout        Kind = "timeout"
	KindAuthentication Kind = "authentication"
	KindProtocol       Kind = "protocol"
	KindResource       Kind = "resource-exhausted"
	KindClosed         Kind = "closed"
	KindInternal       Kind = "internal"
)

// Error adds stable context without requiring callers to parse text.
type Error struct {
	Op        string
	Component string
	Kind      Kind
	Retryable bool
	Err       error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	base := fmt.Sprintf("%s %s: %s", e.Component, e.Op, e.Kind)
	if e.Err != nil {
		return base + ": " + e.Err.Error()
	}
	return base
}

func (e *Error) Unwrap() error { return e.Err }

// IsKind reports whether err or one of its wrapped errors has kind.
func IsKind(err error, kind Kind) bool {
	var target *Error
	return errors.As(err, &target) && target.Kind == kind
}

// New constructs a typed error.
func New(component, op string, kind Kind, retryable bool, cause error) error {
	return &Error{Op: op, Component: component, Kind: kind, Retryable: retryable, Err: cause}
}
