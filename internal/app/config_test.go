package app

import (
	"bytes"
	"testing"
)

func TestParseRunConfigModesAndConflicts(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantMode  ExecMode
		wantError bool
	}{
		{name: "repl", args: []string{}, wantMode: ModeREPL},
		{name: "one shot", args: []string{"-c", "version"}, wantMode: ModeOneShot},
		{name: "script", args: []string{"script.txt"}, wantMode: ModeScript},
		{name: "ui reserved", args: []string{"-ui"}, wantMode: ModeUIReserved},
		{name: "conflict c and script", args: []string{"-c", "version", "script.txt"}, wantError: true},
		{name: "conflict ui and c", args: []string{"-ui", "-c", "version"}, wantError: true},
		{name: "conflict ui and script", args: []string{"-ui", "script.txt"}, wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseRunConfig(tc.args, bytes.NewBuffer(nil), bytes.NewBuffer(nil), bytes.NewBuffer(nil))
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := ResolveMode(cfg); got != tc.wantMode {
				t.Fatalf("mode mismatch: got=%s want=%s", got, tc.wantMode)
			}
		})
	}
}
