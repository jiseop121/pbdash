package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jiseop121/pbdash/internal/cli"
)

const Version = "0.4.1"

type modeResult struct {
	err             error
	alreadyReported bool
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := ParseRunConfig(args, stdin, stdout, stderr)
	if err != nil {
		writeError(stderr, err)
		return MapErrorToExitCode(err)
	}

	dataDir := defaultDataDir()
	dispatcher := cli.NewDispatcher(cli.DispatcherConfig{
		Stdout:  cfg.Stdout,
		Version: Version,
		DataDir: dataDir,
	})
	for _, startupErr := range dispatcher.StartupErrors() {
		writeError(cfg.Stderr, startupErr)
	}

	result := modeResult{}
	mode := ResolveMode(cfg)
	switch mode {
	case ModeUIReserved:
		result.err = NewInvalidArgsError("UI mode is not available in Track 1.", "")
	case ModeOneShot:
		result.err = runOneShot(ctx, cfg.CommandText, dispatcher)
	case ModeScript:
		result = runScript(ctx, cfg.ScriptPath, cfg.Stderr, dispatcher)
	case ModeREPL:
		result = runREPL(ctx, cfg.Stdin, cfg.Stdout, cfg.Stderr, dataDir, dispatcher)
	default:
		result.err = NewRuntimeError("Could not resolve execution mode.", "", nil)
	}

	if result.err != nil && !result.alreadyReported {
		writeError(cfg.Stderr, result.err)
	}
	return MapErrorToExitCode(result.err)
}

func runOneShot(ctx context.Context, commandText string, dispatcher *cli.Dispatcher) error {
	return dispatcher.Execute(ctx, commandText)
}

func runScript(ctx context.Context, path string, stderr io.Writer, dispatcher *cli.Dispatcher) modeResult {
	var lastErr error

	err := cli.RunScript(ctx, path, func(lineNo int, line string) error {
		execErr := dispatcher.Execute(ctx, line)
		if execErr == nil {
			return nil
		}
		if errors.Is(execErr, cli.ErrExitRequested) {
			return cli.ErrExitRequested
		}
		wrapped := WrapScriptLineError(lineNo, execErr)
		writeError(stderr, wrapped)
		lastErr = execErr
		return nil
	})

	if err != nil {
		if errors.Is(err, cli.ErrExitRequested) {
			if lastErr == nil {
				return modeResult{}
			}
			return modeResult{err: lastErr, alreadyReported: true}
		}
		return modeResult{err: err}
	}
	if lastErr == nil {
		return modeResult{}
	}
	return modeResult{err: lastErr, alreadyReported: true}
}

func runREPL(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer, dataDir string, dispatcher *cli.Dispatcher) modeResult {
	var lastErr error

	isTTY := cli.IsTTY(stdin, stdout)
	dispatcher.SetREPLRuntime(true, isTTY)
	err := cli.RunREPLWithConfig(ctx, cli.REPLConfig{
		Stdin:       stdin,
		Stdout:      stdout,
		HistoryFile: filepath.Join(dataDir, "history"),
		Complete:    dispatcher.Complete,
		Execute: func(line string) error {
			execErr := dispatcher.Execute(ctx, line)
			if execErr == nil {
				return nil
			}
			if errors.Is(execErr, cli.ErrExitRequested) {
				return execErr
			}
			writeError(stderr, execErr)
			lastErr = execErr
			return nil
		},
	})

	if err != nil {
		if errors.Is(err, cli.ErrExitRequested) {
			return modeResult{}
		}
		return modeResult{err: err}
	}
	if lastErr == nil {
		return modeResult{}
	}
	return modeResult{err: lastErr, alreadyReported: true}
}

func writeError(stderr io.Writer, err error) {
	if stderr == nil || err == nil {
		return
	}
	_, _ = fmt.Fprintln(stderr, FormatErrorOutput(err))
}

func defaultDataDir() string {
	if custom := os.Getenv("PBDASH_HOME"); custom != "" {
		return custom
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pbdash"
	}
	return filepath.Join(home, ".pbdash")
}
