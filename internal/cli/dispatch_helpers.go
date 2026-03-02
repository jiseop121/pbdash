package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"multi-pocketbase-ui/internal/apperr"
	"multi-pocketbase-ui/internal/pocketbase"
	"multi-pocketbase-ui/internal/storage"
)

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func invalidFlagError(err error) error {
	if err == nil {
		return nil
	}
	return apperr.Invalid("Invalid command arguments.", err.Error())
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	var validationErr *storage.ValidationError
	if errors.As(err, &validationErr) {
		return apperr.Invalid(validationErr.Message, "")
	}
	return apperr.RuntimeErr("Local configuration storage failed.", "Check local file permissions and retry.", err)
}

func mapPBError(err error, superuserAlias, dbAlias string) error {
	if err == nil {
		return nil
	}
	var authErr *pocketbase.AuthError
	if errors.As(err, &authErr) {
		message := strings.TrimSpace(authErr.Message)
		if message == "" {
			message = "Authentication failed."
		}
		if strings.TrimSpace(authErr.Code) != "" {
			message = fmt.Sprintf("%s (code=%s)", message, authErr.Code)
		}
		hint := fmt.Sprintf("Verify credentials for superuser \"%s\" on db \"%s\".", superuserAlias, dbAlias)
		return apperr.ExternalErr(message, hint, err)
	}
	if pocketbase.IsNetworkError(err) {
		return apperr.ExternalErr("Network request to PocketBase failed.", "Check db URL and network connectivity.", err)
	}
	var apiErr *pocketbase.APIError
	if errors.As(err, &apiErr) {
		message := fmt.Sprintf("PocketBase API request failed with status %d.", apiErr.Status)
		if strings.TrimSpace(apiErr.Message) != "" {
			message = fmt.Sprintf("PocketBase API request failed (%d): %s", apiErr.Status, apiErr.Message)
		}
		if strings.TrimSpace(apiErr.Code) != "" {
			message = fmt.Sprintf("%s (code=%s)", message, apiErr.Code)
		}
		return apperr.ExternalErr(message, pbErrorHint(apiErr.Status, apiErr.Code), err)
	}
	return apperr.ExternalErr(err.Error(), "Check connectivity and server status.", err)
}

func pbErrorHint(status int, code string) string {
	switch status {
	case 400:
		if strings.Contains(code, "invalid_") || strings.Contains(code, "missing_") {
			return "Check request options (filter/sort/page/per-page/fields) and retry."
		}
		return "Check request parameters and resource identifiers."
	case 401:
		return "Check superuser credentials and confirm the account can access this target."
	case 403:
		return "The authenticated account lacks permission for this request."
	case 404:
		return "Check collection name, record id, and target db URL."
	default:
		return "Check PocketBase server logs and request parameters."
	}
}

func positiveInt(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, fmt.Errorf("must be positive")
	}
	return v, nil
}
