package app

import (
	"testing"

	"github.com/jiseop121/pbdash/internal/apperr"
)

func TestMapErrorToExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: 0},
		{name: "invalid", err: apperr.Invalid("bad", ""), want: 2},
		{name: "runtime", err: apperr.RuntimeErr("runtime", "", nil), want: 1},
		{name: "external", err: apperr.ExternalErr("external", "", nil), want: 3},
		{name: "script wrapped invalid", err: apperr.WrapScriptLineError(2, apperr.Invalid("bad", "")), want: 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := MapErrorToExitCode(tc.err); got != tc.want {
				t.Fatalf("exit code mismatch: got=%d want=%d", got, tc.want)
			}
		})
	}
}
