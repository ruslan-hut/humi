package entity

import (
	"errors"
	"fmt"
)

// Kinds of failure the client caused. The HTTP layer maps each to a status
// code and shows the message as is.
var (
	ErrInvalid      = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// Error is a client-facing failure of a given kind.
type Error struct {
	Kind error
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func (e *Error) Unwrap() error { return e.Kind }

// Errorf returns a client-facing error of the given kind.
func Errorf(kind error, format string, a ...any) error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, a...)}
}
