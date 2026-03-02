package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

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
		return apperr.ExternalErr("Authentication failed for superuser \""+superuserAlias+"\" on db \""+dbAlias+"\".", "Verify the saved credentials for this superuser alias.", err)
	}
	if pocketbase.IsNetworkError(err) {
		return apperr.ExternalErr("Network request to PocketBase failed.", "Check db URL and network connectivity.", err)
	}
	var apiErr *pocketbase.APIError
	if errors.As(err, &apiErr) {
		return apperr.ExternalErr(fmt.Sprintf("PocketBase API request failed with status %d.", apiErr.Status), "Check credentials, query parameters, and target resource.", err)
	}
	return apperr.ExternalErr("PocketBase request failed.", "Check connectivity and server status.", err)
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
