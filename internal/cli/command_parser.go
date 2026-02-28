package cli

import (
	"fmt"
	"strings"
	"unicode"
)

func ParseCommandLine(line string) ([]string, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, nil
	}

	var (
		tokens       []string
		current      []rune
		inSingle     bool
		inDouble     bool
		escaped      bool
		quoteStarted bool
	)

	flush := func(force bool) {
		if len(current) > 0 || (force && quoteStarted) {
			tokens = append(tokens, string(current))
			current = current[:0]
			quoteStarted = false
		}
	}

	for _, r := range trimmed {
		switch {
		case escaped:
			current = append(current, r)
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			quoteStarted = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
			quoteStarted = true
		case unicode.IsSpace(r) && !inSingle && !inDouble:
			flush(false)
		default:
			current = append(current, r)
		}
	}

	if escaped || inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote or escape sequence")
	}
	flush(false)
	return tokens, nil
}
