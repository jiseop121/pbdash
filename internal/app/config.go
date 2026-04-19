package app

import (
	"io"
	"strings"

	"github.com/jiseop121/pbdash/internal/apperr"
)

type ExecMode string

const (
	ModeTUI        ExecMode = "tui"
	ModeREPL       ExecMode = "repl"
	ModeOneShot    ExecMode = "one-shot"
	ModeScript     ExecMode = "script"
	ModeUIReserved ExecMode = "ui-reserved"
)

type RunConfig struct {
	UIEnabled   bool
	REPLEnabled bool
	CommandText string
	ScriptPath  string
	Stdout      io.Writer
	Stderr      io.Writer
	Stdin       io.Reader
}

func ParseRunConfig(args []string, stdin io.Reader, stdout, stderr io.Writer) (RunConfig, error) {
	cfg := RunConfig{Stdin: stdin, Stdout: stdout, Stderr: stderr}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-ui":
			cfg.UIEnabled = true
		case "-repl":
			cfg.REPLEnabled = true
		case "-c":
			if i+1 >= len(args) {
				return cfg, apperr.Invalid("Missing command text for `-c`.", "Example: pbdash -c \"version\"")
			}
			if strings.TrimSpace(args[i+1]) == "" {
				return cfg, apperr.Invalid("Command text for `-c` cannot be empty.", "Example: pbdash -c \"version\"")
			}
			cfg.CommandText = args[i+1]
			i++
		case "-h", "--help", "help":
			cfg.CommandText = "help"
		case "version", "--version":
			if cfg.CommandText == "" {
				cfg.CommandText = "version"
			}
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return cfg, apperr.Invalid("Unknown option `"+arg+"`.", "Run `pbdash -c \"help\"` to see available commands.")
			}
			if cfg.ScriptPath != "" {
				return cfg, apperr.Invalid("Only one script file path can be provided.", "Use: pbdash <script-file>")
			}
			cfg.ScriptPath = arg
		}
	}

	if err := ValidateRunConfig(cfg); err != nil {
		return RunConfig{}, err
	}
	return cfg, nil
}

func ValidateRunConfig(cfg RunConfig) error {
	if cfg.CommandText != "" && cfg.ScriptPath != "" {
		return apperr.Invalid("Cannot use `-c` and script file path together.", "Choose one mode: `pbdash -c \"...\"` or `pbdash <script-file>`")
	}
	if cfg.UIEnabled && (cfg.CommandText != "" || cfg.ScriptPath != "" || cfg.REPLEnabled) {
		return apperr.Invalid("`-ui` cannot be used with `-c`, script, or `-repl` mode.", "Run `pbdash -ui` alone.")
	}
	if cfg.REPLEnabled && (cfg.CommandText != "" || cfg.ScriptPath != "") {
		return apperr.Invalid("`-repl` cannot be used with `-c` or script mode.", "Run `pbdash -repl` alone.")
	}
	return nil
}

func ResolveMode(cfg RunConfig) ExecMode {
	if cfg.UIEnabled {
		return ModeUIReserved
	}
	if cfg.CommandText != "" {
		return ModeOneShot
	}
	if cfg.ScriptPath != "" {
		return ModeScript
	}
	if cfg.REPLEnabled {
		return ModeREPL
	}
	return ModeTUI
}
