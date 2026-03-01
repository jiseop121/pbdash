package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"multi-pocketbase-ui/internal/apperr"
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
		"pbmulti command reference",
		"Run modes:",
		"Core commands:",
		"DB commands:",
		"Superuser commands:",
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
	}

	for _, token := range required {
		if !strings.Contains(out, token) {
			t.Fatalf("help output missing token %q:\n%s", token, out)
		}
	}
}
