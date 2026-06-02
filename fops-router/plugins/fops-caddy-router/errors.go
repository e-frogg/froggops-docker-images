package fopscaddyrouter

import "fmt"

type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string {
	if e == nil || e.Err == nil {
		return "validation error"
	}

	return e.Err.Error()
}

func (e *ValidationError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func validationErrorf(format string, args ...any) error {
	return &ValidationError{Err: fmt.Errorf(format, args...)}
}
