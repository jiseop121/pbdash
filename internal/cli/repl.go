package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrExitRequested = errors.New("exit requested")

func RunREPL(ctx context.Context, stdin io.Reader, stdout io.Writer, execute func(line string) error) error {
	scanner := bufio.NewScanner(stdin)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, err := fmt.Fprint(stdout, "pbmulti> "); err != nil {
			return err
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			return nil
		}

		err := execute(line)
		if errors.Is(err, ErrExitRequested) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
