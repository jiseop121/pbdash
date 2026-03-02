package app

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func TestVersionMatchesFormulaVersion(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	formulaPath := filepath.Join(repoRoot, "Formula", "pocketbase-multiview.rb")

	data, err := os.ReadFile(formulaPath)
	if err != nil {
		t.Fatalf("read formula: %v", err)
	}

	re := regexp.MustCompile(`version\s+"([^"]+)"`)
	match := re.FindStringSubmatch(string(data))
	if len(match) != 2 {
		t.Fatalf("formula version not found in %s", formulaPath)
	}
	if match[1] != Version {
		t.Fatalf("version mismatch: formula=%s app=%s", match[1], Version)
	}
}
