package cli

import (
	"strings"
	"testing"
)

func TestShouldUseRecordsTUI(t *testing.T) {
	tests := []struct {
		name        string
		view        string
		format      string
		interactive bool
		want        bool
		wantErr     string
	}{
		{name: "auto table interactive", view: "auto", format: "table", interactive: true, want: true},
		{name: "auto table non interactive", view: "auto", format: "table", interactive: false, want: false},
		{name: "auto csv", view: "auto", format: "csv", interactive: true, want: false},
		{name: "table explicit", view: "table", format: "table", interactive: true, want: false},
		{name: "tui interactive", view: "tui", format: "table", interactive: true, want: true},
		{name: "tui non interactive", view: "tui", format: "table", interactive: false, wantErr: "interactive REPL TTY mode"},
		{name: "tui requires table", view: "tui", format: "csv", interactive: true, wantErr: "requires `--format table`"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shouldUseRecordsTUI(tc.view, tc.format, tc.interactive)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error mismatch: got=%q want contains=%q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("result mismatch: got=%v want=%v", got, tc.want)
			}
		})
	}
}
