package apperr

import (
	"errors"
	"fmt"
)

type Kind string

const (
	InvalidArgs Kind = "invalid_args"
	Runtime     Kind = "runtime"
	External    Kind = "external"
)

type Error struct {
	Kind    Kind
	Message string
	Hint    string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(kind Kind, message, hint string, cause error) *Error {
	return &Error{Kind: kind, Message: message, Hint: hint, Cause: cause}
}

func Invalid(message, hint string) *Error {
	return New(InvalidArgs, message, hint, nil)
}

func RuntimeErr(message, hint string, cause error) *Error {
	return New(Runtime, message, hint, cause)
}

func ExternalErr(message, hint string, cause error) *Error {
	return New(External, message, hint, cause)
}

type ScriptLineError struct {
	Line int
	Err  error
}

func (e *ScriptLineError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("Script failed at line %d: %s", e.Line, RootMessage(e.Err))
}

func (e *ScriptLineError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func WrapScriptLineError(line int, err error) error {
	if err == nil {
		return nil
	}
	return &ScriptLineError{Line: line, Err: err}
}

func RootMessage(err error) string {
	if err == nil {
		return ""
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	var scriptErr *ScriptLineError
	if errors.As(err, &scriptErr) {
		return scriptErr.Error()
	}
	return err.Error()
}

func Format(err error) string {
	if err == nil {
		return ""
	}

	var scriptErr *ScriptLineError
	if errors.As(err, &scriptErr) {
		return fmt.Sprintf("Error: Script failed at line %d: %s", scriptErr.Line, RootMessage(scriptErr.Err))
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		if appErr.Hint == "" {
			return fmt.Sprintf("Error: %s", appErr.Message)
		}
		return fmt.Sprintf("Error: %s\nHint: %s", appErr.Message, appErr.Hint)
	}

	return fmt.Sprintf("Error: %s", err.Error())
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		switch appErr.Kind {
		case InvalidArgs:
			return 2
		case External:
			return 3
		case Runtime:
			return 1
		default:
			return 1
		}
	}

	var scriptErr *ScriptLineError
	if errors.As(err, &scriptErr) {
		return ExitCode(scriptErr.Err)
	}

	return 1
}
