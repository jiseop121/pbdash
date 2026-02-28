//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPBMultiRealPocketBaseSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skip e2e test in short mode")
	}

	pocketbaseBin := envOrDefault("POCKETBASE_BIN", "pocketbase")
	if _, err := exec.LookPath(pocketbaseBin); err != nil {
		t.Fatalf("pocketbase binary not found: set POCKETBASE_BIN (err=%v)", err)
	}

	repoRoot := mustRepoRoot(t)
	workDir := t.TempDir()
	pbWorkDir := filepath.Join(workDir, "pocketbase")
	if err := os.MkdirAll(pbWorkDir, 0o755); err != nil {
		t.Fatalf("mkdir pocketbase workdir: %v", err)
	}

	const (
		email = "root@example.com"
		pass  = "pass123456"
	)
	if err := createSuperuser(pocketbaseBin, pbWorkDir, email, pass); err != nil {
		t.Fatalf("create superuser failed: %v", err)
	}

	port, err := findFreePort()
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	serveLogs := &bytes.Buffer{}
	serveCtx, cancelServe := context.WithCancel(context.Background())
	serveCmd := exec.CommandContext(serveCtx, pocketbaseBin, "serve", fmt.Sprintf("--http=127.0.0.1:%d", port))
	serveCmd.Dir = pbWorkDir
	serveCmd.Stdout = serveLogs
	serveCmd.Stderr = serveLogs
	if err := serveCmd.Start(); err != nil {
		t.Fatalf("start pocketbase serve: %v", err)
	}
	defer stopProcess(cancelServe, serveCmd)

	if err := waitForPocketBase(baseURL, 20*time.Second); err != nil {
		t.Fatalf("pocketbase startup failed: %v\nlogs:\n%s", err, serveLogs.String())
	}

	pbmultiBin := filepath.Join(workDir, "pbmulti")
	_ = runCommand(t, repoRoot, nil, "go", "build", "-o", pbmultiBin, "./cmd/pbmulti")

	pbmultiHome := filepath.Join(workDir, "pbmulti-home")
	env := append(os.Environ(), "PBMULTI_HOME="+pbmultiHome)

	runCLI := func(line string) string {
		return runCommand(t, repoRoot, env, pbmultiBin, "-c", line)
	}

	versionOut := strings.TrimSpace(runCLI("version"))
	if versionOut == "" {
		t.Fatalf("version output mismatch: %q", versionOut)
	}

	helpOut := runCLI("help")
	if !strings.Contains(helpOut, "api record") {
		t.Fatalf("help output missing api command: %q", helpOut)
	}

	_ = runCLI(fmt.Sprintf("db add --alias local --url %s", baseURL))
	dbListOut := runCLI("db list")
	if !strings.Contains(dbListOut, "local") {
		t.Fatalf("db list missing alias: %q", dbListOut)
	}

	_ = runCLI(fmt.Sprintf("superuser add --db local --alias root --email %s --password %s", email, pass))
	suListOut := runCLI("superuser list --db local")
	if !strings.Contains(suListOut, email) {
		t.Fatalf("superuser list missing email: %q", suListOut)
	}

	collectionsOut := runCLI("api collections --db local --superuser root")
	if !strings.Contains(collectionsOut, "_superusers") {
		t.Fatalf("collections output missing _superusers: %q", collectionsOut)
	}

	collectionOut := runCLI("api collection --db local --superuser root --name _superusers")
	if !strings.Contains(collectionOut, "_superusers") {
		t.Fatalf("collection output missing _superusers: %q", collectionOut)
	}

	recordsCSVPath := filepath.Join(workDir, "superusers.csv")
	recordsOut := runCLI(fmt.Sprintf("api records --db local --superuser root --collection _superusers --page 1 --per-page 10 --sort -created --filter id!=\"\" --format csv --out '%s'", recordsCSVPath))
	if !strings.Contains(recordsOut, "Exported") {
		t.Fatalf("records csv export summary missing: %q", recordsOut)
	}
	recordID := mustExtractFirstRecordID(t, recordsCSVPath)

	recordMDPath := filepath.Join(workDir, "superuser.md")
	recordOut := runCLI(fmt.Sprintf("api record --db local --superuser root --collection _superusers --id %s --format markdown --out '%s'", recordID, recordMDPath))
	if !strings.Contains(recordOut, "Exported 1 rows") {
		t.Fatalf("record markdown export summary mismatch: %q", recordOut)
	}
	recordMD, err := os.ReadFile(recordMDPath)
	if err != nil {
		t.Fatalf("read record markdown: %v", err)
	}
	if !strings.Contains(string(recordMD), recordID) {
		t.Fatalf("record markdown does not include id=%s", recordID)
	}

	scriptPath := filepath.Join(workDir, "smoke.script")
	scriptBody := "# script smoke\nversion\ndb list\n"
	if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o644); err != nil {
		t.Fatalf("write script file: %v", err)
	}
	scriptOut := runCommand(t, repoRoot, env, pbmultiBin, scriptPath)
	if !strings.Contains(scriptOut, versionOut) || !strings.Contains(scriptOut, "local") {
		t.Fatalf("script mode output mismatch: %q", scriptOut)
	}

	_ = runCLI("superuser remove --db local --alias root")
	_ = runCLI("db remove --alias local")
	finalDBList := runCLI("db list")
	if strings.Contains(finalDBList, "local") {
		t.Fatalf("db alias should be removed: %q", finalDBList)
	}
}

func createSuperuser(pocketbaseBin, dir, email, pass string) error {
	candidates := [][]string{
		{"superuser", "create", email, pass},
		{"superuser", "upsert", email, pass},
		{"admin", "create", email, pass},
		{"admins", "create", email, pass},
	}

	errorsSeen := make([]string, 0, len(candidates))
	for _, args := range candidates {
		cmd := exec.Command(pocketbaseBin, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		errorsSeen = append(errorsSeen, fmt.Sprintf("%s -> %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out))))
	}
	return fmt.Errorf("all create commands failed: %s", strings.Join(errorsSeen, " | "))
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve runtime caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
}

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func waitForPocketBase(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}
	var lastErr error

	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/api/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
			lastErr = fmt.Errorf("health endpoint status=%d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(250 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = errors.New("unknown startup error")
	}
	return lastErr
}

func stopProcess(cancel context.CancelFunc, cmd *exec.Cmd) {
	cancel()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-waitDone
	}
}

func runCommand(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s %s\nerr: %v\noutput:\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func mustExtractFirstRecordID(t *testing.T, csvPath string) string {
	t.Helper()
	f, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("open csv: %v", err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("csv has no data rows: %v", rows)
	}

	idColumn := -1
	for i, col := range rows[0] {
		if col == "id" {
			idColumn = i
			break
		}
	}
	if idColumn < 0 {
		t.Fatalf("csv header does not include id: %v", rows[0])
	}
	id := strings.TrimSpace(rows[1][idColumn])
	if id == "" {
		t.Fatalf("first data row id is empty: %v", rows[1])
	}
	return id
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
