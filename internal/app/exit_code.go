package app

import "multi-pocketbase-ui/internal/apperr"

func MapErrorToExitCode(err error) int {
	return apperr.ExitCode(err)
}
