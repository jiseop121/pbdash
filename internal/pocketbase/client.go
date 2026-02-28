package pocketbase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("pocketbase api error (status=%d)", e.Status)
}

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	if e == nil || e.Message == "" {
		return "authentication failed"
	}
	return e.Message
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
	targets := []string{
		"/api/collections/_superusers/auth-with-password",
		"/api/admins/auth-with-password",
	}

	payload := map[string]string{"identity": email, "password": password}
	body, _ := json.Marshal(payload)

	var lastErr error
	for _, endpoint := range targets {
		u, err := joinURL(baseURL, endpoint)
		if err != nil {
			return "", err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return "", &AuthError{Message: "authentication failed"}
		}
		if resp.StatusCode >= 400 {
			lastErr = &APIError{Status: resp.StatusCode, Body: string(respBody)}
			continue
		}

		var parsed map[string]any
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return "", err
		}
		token, _ := parsed["token"].(string)
		if strings.TrimSpace(token) == "" {
			return "", errors.New("authentication token is empty")
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{Message: "authentication failed"}
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{Status: resp.StatusCode, Body: string(body)}
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err == nil {
		return parsed, nil
	}
	var list []any
	if err := json.Unmarshal(body, &list); err == nil {
		return map[string]any{"items": list}, nil
	}
	return nil, errors.New("unsupported pocketbase response body")
}

func joinURL(baseURL, endpoint string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	base.Path = path.Join(base.Path, endpoint)
	return base.String(), nil
}
