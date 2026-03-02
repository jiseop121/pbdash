package cli

import (
	"bytes"
	"testing"
)

func TestCompleteTopLevelCommands(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	got := d.Complete("")
	mustContain(t, got, "db")
	mustContain(t, got, "api")
	mustContain(t, got, "context")
}

func TestCompleteSuggestsDBAndSuperuserAliases(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("dev", "http://127.0.0.1:8090"); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if err := d.suStore.Add("dev", "root", "root@example.com", "pw"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	dbSuggestions := d.Complete("api records --db ")
	mustContain(t, dbSuggestions, "dev")

	suSuggestions := d.Complete("api records --db dev --superuser ")
	mustContain(t, suSuggestions, "root")
}

func TestCompleteMatchesPrefixCaseInsensitively(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	if err := d.dbStore.Add("Prod", "http://127.0.0.1:8090"); err != nil {
		t.Fatalf("add db: %v", err)
	}

	got := d.Complete("api records --db p")
	mustContain(t, got, "Prod")
}

func TestCompleteSuggestsViewModes(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	got := d.Complete("api records --view ")
	mustContain(t, got, "auto")
	mustContain(t, got, "tui")
	mustContain(t, got, "table")
}

func mustContain(t *testing.T, items []string, want string) {
	t.Helper()
	for _, it := range items {
		if it == want {
			return
		}
	}
	t.Fatalf("missing %q in %v", want, items)
}
