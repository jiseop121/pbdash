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

func TestRunParseErrorWritesToStderr(t *testing.T) {
	stdin := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{"--unknown"}, stdin, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code mismatch: got=%d want=2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout")
	}
	if !strings.Contains(stderr.String(), "Unknown option `--unknown`.") {
		t.Fatalf("missing parse error output: %s", stderr.String())
	}
}

func TestRunEmptyOneShotCommandReturnsInvalidArgs(t *testing.T) {
	stdin := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{"-c", ""}, stdin, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code mismatch: got=%d want=2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout")
	}
	if !strings.Contains(stderr.String(), "Command text for `-c` cannot be empty.") {
		t.Fatalf("missing empty one-shot error output: %s", stderr.String())
	}
}

func TestRunREPLContinuesAfterCommandError(t *testing.T) {
	stdin := bytes.NewBufferString("version\nunknown-command\nversion\nexit\n")
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{}, stdin, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code mismatch: got=%d want=2", code)
	}
	if count := strings.Count(stdout.String(), Version); count != 2 {
		t.Fatalf("repl should continue after error, version output count=%d, stdout=%q", count, stdout.String())
	}
	if !strings.Contains(stderr.String(), "Unknown command `unknown-command`.") {
		t.Fatalf("missing repl error output: %s", stderr.String())
	}
}

func TestRunREPLStopsOnPrefixedExitCommand(t *testing.T) {
	stdin := bytes.NewBufferString("version\npbviewer exit\nversion\n")
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{}, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code mismatch: got=%d want=0", code)
	}
	if count := strings.Count(stdout.String(), Version); count != 1 {
		t.Fatalf("repl should stop at prefixed exit command, version output count=%d, stdout=%q", count, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}
}

func TestRunScriptContinuesAfterCommandError(t *testing.T) {
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
	if count := strings.Count(stdout.String(), Version); count != 2 {
		t.Fatalf("script should continue after error, version output count=%d, stdout=%q", count, stdout.String())
	}
	if !strings.Contains(stderr.String(), "Script failed at line 2") {
		t.Fatalf("missing script line error: %s", stderr.String())
	}
}

func TestRunScriptUsesLastErrorExitCode(t *testing.T) {
	t.Setenv("PBVIEWER_HOME", t.TempDir())

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "script.txt")
	content := strings.Join([]string{
		"unknown-command",
		"db add --alias dead --url http://127.0.0.1:1",
		"superuser add --db dead --alias root --email root@example.com --password pass123456",
		"api collections --db dead --superuser root",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	stdin := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	code := Run(context.Background(), []string{scriptPath}, stdin, stdout, stderr)
	if code != 3 {
		t.Fatalf("exit code mismatch: got=%d want=3", code)
	}
	if !strings.Contains(stderr.String(), "Script failed at line 1") {
		t.Fatalf("missing first script error output: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Script failed at line 4") {
		t.Fatalf("missing last script error output: %s", stderr.String())
	}
}

func TestRunScriptExitStopsFurtherCommands(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "script.txt")
	content := "unknown-command\nexit\nversion\n"
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
	if strings.Contains(stdout.String(), Version) {
		t.Fatalf("script should stop at exit command, stdout=%q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Script failed at line 1") {
		t.Fatalf("missing script error output before exit: %s", stderr.String())
	}
}
