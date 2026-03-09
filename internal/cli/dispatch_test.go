package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiseop121/pbdash/internal/apperr"
)

func TestAPICSVRequiresOut(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})
	_ = d.dbStore.Add("dev", "http://127.0.0.1:8090")
	_ = d.suStore.Add("dev", "root", "root@example.com", "pw")

	err := d.Execute(context.Background(), "api collections --db dev --superuser root --format csv")
	if err == nil {
		t.Fatalf("expected error")
	}
	if apperr.ExitCode(err) != 2 {
		t.Fatalf("exit code mismatch: got=%d want=2", apperr.ExitCode(err))
	}
	if !strings.Contains(apperr.Format(err), "Missing required option `--out`") {
		t.Fatalf("unexpected error message: %s", apperr.Format(err))
	}
}

func TestAPIRecordsQueryOptionsAndMetaOutput(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/collections/_superusers/auth-with-password":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "tok"})
		case "/api/collections/posts/records":
			gotQuery = r.URL.Query()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{"id": "1", "title": "hello", "created": "2026-02-28T08:12:00Z"}},
				"page":  1, "perPage": 2, "totalItems": 24, "totalPages": 12,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("dev", server.URL); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	cmd := "api records --db dev --superuser root --collection posts --page 1 --per-page 2 --sort -created --filter status='open'"
	if err := d.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotQuery.Get("page") != "1" || gotQuery.Get("perPage") != "2" {
		t.Fatalf("missing paging query: %v", gotQuery)
	}
	if gotQuery.Get("sort") != "-created" {
		t.Fatalf("missing sort query: %v", gotQuery)
	}
	if gotQuery.Get("filter") != "status='open'" {
		t.Fatalf("missing filter query: %v", gotQuery)
	}
	out := buf.String()
	if !strings.Contains(out, "page=1 perPage=2 totalItems=24 totalPages=12") {
		t.Fatalf("missing meta output: %s", out)
	}
	if !strings.Contains(out, "1 rows") {
		t.Fatalf("missing rows output: %s", out)
	}
}

func TestAPIAuthFailureMapsToExternalExitCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/collections/_superusers/auth-with-password" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("dev", server.URL); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	err := d.Execute(context.Background(), "api collections --db dev --superuser root")
	if err == nil {
		t.Fatalf("expected auth error")
	}
	if apperr.ExitCode(err) != 3 {
		t.Fatalf("exit code mismatch: got=%d want=3", apperr.ExitCode(err))
	}
	if !strings.Contains(apperr.Format(err), "Authentication failed") {
		t.Fatalf("unexpected error format: %s", apperr.Format(err))
	}
}

func TestHelpIncludesCommandDescriptions(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})

	if err := d.Execute(context.Background(), "help"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	required := []string{
		"pbdash command reference",
		"Run modes:",
		"Core commands:",
		"DB commands:",
		"Superuser commands:",
		"Context commands:",
		"API commands (read-only GET):",
		"version                         Print CLI version.",
		"help                            Show available commands.",
		"db add --alias <dbAlias> --url <baseUrl>",
		"Save a PocketBase base URL as a db alias.",
		"superuser add --db <dbAlias> --alias <superuserAlias> --email <email> --password <password>",
		"Save superuser credentials for a db alias.",
		"api record --db <dbAlias> --superuser <superuserAlias> --collection <collectionName> --id <recordId>",
		"Get one record by id.",
		"csv/markdown requires --out <path>.",
		"pbdash                         Start full-screen TUI mode.",
		"pbdash -repl                   Start legacy REPL mode.",
		"pbdash -ui                     Reserved for the future web UI (currently under development).",
		"TUI view requires a TTY terminal.",
	}

	for _, token := range required {
		if !strings.Contains(out, token) {
			t.Fatalf("help output missing token %q:\n%s", token, out)
		}
	}
}

func TestContextUseAndAPIFallbackTargetResolution(t *testing.T) {
	server := newPocketBaseReadServer(t, "tok", nil)
	defer server.Close()

	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("dev", server.URL); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	if err := d.Execute(context.Background(), "context use --db dev --superuser root"); err != nil {
		t.Fatalf("context use: %v", err)
	}
	if err := d.Execute(context.Background(), "api records --collection posts --view table"); err != nil {
		t.Fatalf("api records with context fallback: %v", err)
	}
	if !strings.Contains(buf.String(), "Updated session context") {
		t.Fatalf("expected context update output: %s", buf.String())
	}
}

func TestContextSaveAndLoadAcrossDispatcherInstances(t *testing.T) {
	dataDir := t.TempDir()
	server := newPocketBaseReadServer(t, "tok", nil)
	defer server.Close()

	d1 := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: dataDir})
	if err := d1.dbStore.Add("dev", server.URL); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d1.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}
	if err := d1.Execute(context.Background(), "context use --db dev --superuser root"); err != nil {
		t.Fatalf("context use: %v", err)
	}
	if err := d1.Execute(context.Background(), "context save"); err != nil {
		t.Fatalf("context save: %v", err)
	}

	buf := bytes.NewBuffer(nil)
	d2 := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: dataDir})
	if err := d2.Execute(context.Background(), "api collections"); err != nil {
		t.Fatalf("api collections with saved context: %v", err)
	}
}

func TestNewDispatcherReportsContextLoadError(t *testing.T) {
	dataDir := t.TempDir()
	contextPath := filepath.Join(dataDir, "context.json")
	if err := os.WriteFile(contextPath, []byte("{broken json"), 0o600); err != nil {
		t.Fatalf("write broken context: %v", err)
	}

	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: dataDir})
	errs := d.StartupErrors()
	if len(errs) != 1 {
		t.Fatalf("startup error count mismatch: got=%d want=1", len(errs))
	}
	formatted := apperr.Format(errs[0])
	if !strings.Contains(formatted, "Could not load saved default context.") {
		t.Fatalf("missing startup error message: %s", formatted)
	}
	if !strings.Contains(formatted, contextPath) {
		t.Fatalf("missing context path in startup hint: %s", formatted)
	}
}

func TestAPIRecordsViewTUIRequiresInteractiveTTY(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("dev", "http://127.0.0.1:8090"); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	err := d.Execute(context.Background(), "api records --db dev --superuser root --collection posts --view tui")
	if err == nil {
		t.Fatalf("expected error")
	}
	if apperr.ExitCode(err) != 2 {
		t.Fatalf("exit code mismatch: got=%d want=2", apperr.ExitCode(err))
	}
	if !strings.Contains(apperr.Format(err), "requires a TTY terminal") {
		t.Fatalf("unexpected error: %s", apperr.Format(err))
	}
}

func TestAPITokenCacheReusesTokenForSameTarget(t *testing.T) {
	var authCalls int32
	server := newPocketBaseReadServer(t, "tok", &authCalls)
	defer server.Close()

	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("dev", server.URL); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	cmd := "api records --db dev --superuser root --collection posts --view table"
	if err := d.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if err := d.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if got := atomic.LoadInt32(&authCalls); got != 1 {
		t.Fatalf("auth call count mismatch: got=%d want=1", got)
	}
}

func TestAPITokenCacheRetriesOn401(t *testing.T) {
	var authCalls int32
	var firstTokenSeen int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/collections/_superusers/auth-with-password":
			call := atomic.AddInt32(&authCalls, 1)
			if call == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{"token": "first.token.value"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "second.token.value"})
			return
		case "/api/collections/posts/records":
			auth := r.Header.Get("Authorization")
			if strings.Contains(auth, "first.token.value") && atomic.CompareAndSwapInt32(&firstTokenSeen, 0, 1) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{"id": "1"}},
				"page":  1, "perPage": 1, "totalItems": 1, "totalPages": 1,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	buf := bytes.NewBuffer(nil)
	d := NewDispatcher(DispatcherConfig{Stdout: buf, Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("dev", server.URL); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	if err := d.Execute(context.Background(), "api records --db dev --superuser root --collection posts --view table"); err != nil {
		t.Fatalf("api records failed: %v", err)
	}
	if got := atomic.LoadInt32(&authCalls); got != 2 {
		t.Fatalf("auth call count mismatch: got=%d want=2", got)
	}
}

func newPocketBaseReadServer(t *testing.T, token string, authCalls *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/collections/_superusers/auth-with-password":
			if authCalls != nil {
				atomic.AddInt32(authCalls, 1)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"token": token})
		case "/api/collections":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"id": "col1", "name": "posts"}}})
		case "/api/collections/posts/records":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{"id": "1", "title": "hello"}},
				"page":  1, "perPage": 20, "totalItems": 1, "totalPages": 1,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
