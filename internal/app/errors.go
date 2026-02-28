package app

import "multi-pocketbase-ui/internal/apperr"

type AppErrorKind = apperr.Kind

type AppError = apperr.Error

type ScriptLineError = apperr.ScriptLineError

const (
	ErrInvalidArgs AppErrorKind = apperr.InvalidArgs
	ErrRuntime     AppErrorKind = apperr.Runtime
	ErrExternal    AppErrorKind = apperr.External
)

func NewInvalidArgsError(message, hint string) *AppError {
	return apperr.Invalid(message, hint)
}

func NewRuntimeError(message, hint string, cause error) *AppError {
	return apperr.RuntimeErr(message, hint, cause)
}

func NewExternalError(message, hint string, cause error) *AppError {
	return apperr.ExternalErr(message, hint, cause)
}

func WrapScriptLineError(line int, err error) error {
	return apperr.WrapScriptLineError(line, err)
}

func FormatErrorOutput(err error) string {
	return apperr.Format(err)
}
