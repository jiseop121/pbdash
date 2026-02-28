package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunScriptSkipsBlankAndComment(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "script.txt")
	data := "\n# comment\nversion\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	count := 0
	err := RunScript(context.Background(), path, func(lineNo int, line string) error {
		count++
		if lineNo != 3 {
			t.Fatalf("line number mismatch: got=%d want=3", lineNo)
		}
		if line != "version" {
			t.Fatalf("line mismatch: %q", line)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("execution count mismatch: got=%d want=1", count)
	}
}

func TestRunScriptFailFast(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "script.txt")
	data := "version\nfail\nversion\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	count := 0
	err := RunScript(context.Background(), path, func(lineNo int, line string) error {
		count++
		if line == "fail" {
			return os.ErrInvalid
		}
		return nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if count != 2 {
		t.Fatalf("expected fail-fast at second command, got count=%d", count)
	}
}
