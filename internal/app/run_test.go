package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUIReserved(t *testing.T) {
	stdin := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{"-ui"}, stdin, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code mismatch: got=%d want=2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout")
	}
	if !strings.Contains(stderr.String(), "Error: UI mode is not available in Track 1.") {
		t.Fatalf("missing ui error message: %s", stderr.String())
	}
}

func TestRunStdoutStderrSeparation(t *testing.T) {
	stdin := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{"-c", "version"}, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code mismatch: got=%d want=0", code)
	}
	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatalf("expected version output on stdout")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr on success: %s", stderr.String())
	}
}

func TestRunScriptFailFast(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "script.txt")
	content := "version\nunknown-command\nversion\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	stdin := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{scriptPath}, stdin, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code mismatch: got=%d want=2", code)
	}
	if count := strings.Count(stdout.String(), Version); count != 1 {
		t.Fatalf("script did not fail-fast, version output count=%d, stdout=%q", count, stdout.String())
	}
	if !strings.Contains(stderr.String(), "Script failed at line 2") {
		t.Fatalf("missing script line error: %s", stderr.String())
	}
}
