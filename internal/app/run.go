package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"multi-pocketbase-ui/internal/cli"
)

const Version = "0.1.0"

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := ParseRunConfig(args, stdin, stdout, stderr)
	if err != nil {
		writeError(cfg.Stderr, err)
		return MapErrorToExitCode(err)
	}

	dispatcher := cli.NewDispatcher(cli.DispatcherConfig{
		Stdout:  cfg.Stdout,
		Version: Version,
		DataDir: defaultDataDir(),
	})

	mode := ResolveMode(cfg)
	switch mode {
	case ModeUIReserved:
		err = NewInvalidArgsError("UI mode is not available in Track 1.", "")
	case ModeOneShot:
		err = runOneShot(ctx, cfg.CommandText, dispatcher)
	case ModeScript:
		err = runScript(ctx, cfg.ScriptPath, dispatcher)
	case ModeREPL:
		err = runREPL(ctx, cfg.Stdin, cfg.Stdout, dispatcher)
	default:
		err = NewRuntimeError("Could not resolve execution mode.", "", nil)
	}

	if err != nil {
		writeError(cfg.Stderr, err)
	}
	return MapErrorToExitCode(err)
}

func runOneShot(ctx context.Context, commandText string, dispatcher *cli.Dispatcher) error {
	return dispatcher.Execute(ctx, commandText)
}

func runScript(ctx context.Context, path string, dispatcher *cli.Dispatcher) error {
	return cli.RunScript(ctx, path, func(lineNo int, line string) error {
		err := dispatcher.Execute(ctx, line)
		if err != nil {
			return WrapScriptLineError(lineNo, err)
		}
		return nil
	})
}

func runREPL(ctx context.Context, stdin io.Reader, stdout io.Writer, dispatcher *cli.Dispatcher) error {
	return cli.RunREPL(ctx, stdin, stdout, func(line string) error {
		return dispatcher.Execute(ctx, line)
	})
}

func writeError(stderr io.Writer, err error) {
	if stderr == nil || err == nil {
		return
	}
	_, _ = fmt.Fprintln(stderr, FormatErrorOutput(err))
}

func defaultDataDir() string {
	if custom := os.Getenv("PBMULTI_HOME"); custom != "" {
		return custom
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pbmulti"
	}
	return filepath.Join(home, ".pbmulti")
}
