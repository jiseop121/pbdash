package cli

import (
	"bufio"
	"context"
	"os"
	"strings"
	"unicode/utf8"

	"multi-pocketbase-ui/internal/apperr"
)

type ScriptLineExecutor func(lineNo int, line string) error

func RunScript(ctx context.Context, path string, execute ScriptLineExecutor) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return apperr.RuntimeErr("Could not read script file \""+path+"\".", "Check file path and read permission.", err)
	}
	if !utf8.Valid(data) {
		return apperr.RuntimeErr("Script file must be UTF-8 encoded.", "Re-save the script file using UTF-8 encoding.", nil)
	}

	lineNo := 0
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return apperr.RuntimeErr("Script execution was interrupted.", "", ctx.Err())
		default:
		}

		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := execute(lineNo, line); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return apperr.RuntimeErr("Could not read script content.", "", err)
	}
	return nil
}
