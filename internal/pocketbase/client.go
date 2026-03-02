package pocketbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	pbclient "github.com/mrchypark/pocketbase-client"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

type APIError struct {
	Status  int
	Code    string
	Message string
	Body    string
	Cause   error
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		if strings.TrimSpace(e.Code) != "" {
			return fmt.Sprintf("pocketbase api error (status=%d code=%s): %s", e.Status, e.Code, e.Message)
		}
		return fmt.Sprintf("pocketbase api error (status=%d): %s", e.Status, e.Message)
	}
	return fmt.Sprintf("pocketbase api error (status=%d)", e.Status)
}

func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Cause != nil {
		return e.Cause
	}
	return nil
}

type AuthError struct {
	Status  int
	Code    string
	Message string
	Cause   error
}

func (e *AuthError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) == "" {
		return "authentication failed"
	}
	return e.Message
}

func (e *AuthError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Cause != nil {
		return e.Cause
	}
	return nil
}

func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

func (c *Client) Authenticate(ctx context.Context, baseURL, email, password string) (string, error) {
	sdkClient := c.newSDKClient(baseURL)
	payload := map[string]string{"identity": email, "password": password}
	targets := []string{
		"/api/collections/_superusers/auth-with-password",
		"/api/admins/auth-with-password",
	}

	var lastErr error
	for _, endpoint := range targets {
		u, err := joinURL(baseURL, endpoint)
		if err != nil {
			return "", err
		}
		var response map[string]any
		if err := sdkClient.Send(ctx, http.MethodPost, u, payload, &response); err != nil {
			mapped := mapSDKError(err)
			var authErr *AuthError
			if errors.As(mapped, &authErr) {
				return "", mapped
			}
			lastErr = mapped
			continue
		}
		token, _ := response["token"].(string)
		if strings.TrimSpace(token) == "" {
			lastErr = errors.New("authentication token is empty")
			continue
		}
		return token, nil
	}
	if lastErr == nil {
		lastErr = errors.New("authentication failed")
	}
	return "", lastErr
}

func (c *Client) GetJSON(ctx context.Context, baseURL, token, endpoint string, query map[string]string) (map[string]any, error) {
	u, err := joinURL(baseURL, endpoint)
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	q := parsedURL.Query()
	for k, v := range query {
		if strings.TrimSpace(v) != "" {
			q.Set(k, v)
		}
	}
	parsedURL.RawQuery = q.Encode()

	sdkClient := c.newSDKClient(baseURL)
	if formatted := formatTokenForAuthorization(token); formatted != "" {
		sdkClient.WithToken(formatted)
	}

	var raw json.RawMessage
	if err := sdkClient.Send(ctx, http.MethodGet, parsedURL.String(), nil, &raw); err != nil {
		return nil, mapSDKError(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err == nil {
		return parsed, nil
	}
	var list []any
	if err := json.Unmarshal(raw, &list); err == nil {
		return map[string]any{"items": list}, nil
	}
	return nil, errors.New("unsupported pocketbase response body")
}

func (c *Client) newSDKClient(baseURL string) *pbclient.Client {
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return pbclient.NewClient(baseURL, pbclient.WithHTTPClient(httpClient))
}

func mapSDKError(err error) error {
	if err == nil {
		return nil
	}
	var pbErr *pbclient.Error
	if !errors.As(err, &pbErr) {
		return err
	}
	if pbErr.IsAuth() || pbclient.IsAuthenticationFailed(err) {
		return &AuthError{
			Status:  pbErr.Status,
			Code:    pbErr.Code,
			Message: strings.TrimSpace(pbErr.Message),
			Cause:   err,
		}
	}
	return &APIError{
		Status:  pbErr.Status,
		Code:    pbErr.Code,
		Message: strings.TrimSpace(pbErr.Message),
		Cause:   err,
	}
}

func formatTokenForAuthorization(token string) string {
	tok := strings.TrimSpace(token)
	if tok == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(tok), "bearer ") {
		return tok
	}
	return "Bearer " + tok
}

func joinURL(baseURL, endpoint string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	base.Path = path.Join(base.Path, strings.TrimPrefix(endpoint, "/"))
	return base.String(), nil
}
