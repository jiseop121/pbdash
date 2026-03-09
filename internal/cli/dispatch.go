package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/jiseop121/pbdash/internal/apperr"
	"github.com/jiseop121/pbdash/internal/pocketbase"
	"github.com/jiseop121/pbdash/internal/storage"
)

type DispatcherConfig struct {
	Stdout  io.Writer
	Version string
	DataDir string
}

type commandContext struct {
	DBAlias        string
	SuperuserAlias string
}

type authCacheKey struct {
	dbAlias string
	suAlias string
}

type authCacheEntry struct {
	token     string
	expiresAt time.Time
	hasExpiry bool
}

type Dispatcher struct {
	stdout   io.Writer
	version  string
	dbStore  *storage.DBStore
	suStore  *storage.SuperuserStore
	ctxStore *storage.ContextStore
	pbClient *pocketbase.Client

	sessionCtx commandContext
	savedCtx   commandContext
	hasSaved   bool

	isREPL bool
	isTTY  bool

	authCache map[authCacheKey]authCacheEntry
	now       func() time.Time

	navigatorRunner func(context.Context, navigatorRoute) error

	startupErrs []error
}

func NewDispatcher(cfg DispatcherConfig) *Dispatcher {
	d := &Dispatcher{
		stdout:      cfg.Stdout,
		version:     cfg.Version,
		dbStore:     storage.NewDBStore(cfg.DataDir),
		suStore:     storage.NewSuperuserStore(cfg.DataDir),
		ctxStore:    storage.NewContextStore(cfg.DataDir),
		pbClient:    pocketbase.NewClient(),
		authCache:   map[authCacheKey]authCacheEntry{},
		now:         time.Now,
		startupErrs: make([]error, 0),
	}
	d.navigatorRunner = func(ctx context.Context, route navigatorRoute) error {
		return d.runNavigatorTUI(ctx, route)
	}
	if saved, ok, err := d.ctxStore.Load(); err == nil && ok {
		d.savedCtx = commandContext{DBAlias: saved.DBAlias, SuperuserAlias: saved.SuperuserAlias}
		d.hasSaved = true
	} else if err != nil {
		ctxPath := filepath.Join(cfg.DataDir, "context.json")
		d.startupErrs = append(d.startupErrs, apperr.RuntimeErr(
			"Could not load saved default context.",
			fmt.Sprintf("Fix or remove %q, then retry.", ctxPath),
			err,
		))
	}
	return d
}

func (d *Dispatcher) StartupErrors() []error {
	if len(d.startupErrs) == 0 {
		return nil
	}
	out := make([]error, len(d.startupErrs))
	copy(out, d.startupErrs)
	return out
}

func (d *Dispatcher) SetTerminal(isTTY bool) {
	d.isTTY = isTTY
}

func (d *Dispatcher) SetREPLRuntime(isREPL bool) {
	d.isREPL = isREPL
}

func (d *Dispatcher) HasTTY() bool {
	return d.isTTY
}

func (d *Dispatcher) IsInteractiveTTY() bool {
	return d.isREPL && d.isTTY
}

func (d *Dispatcher) RunNavigator(ctx context.Context) error {
	return d.navigatorRunner(ctx, navigatorRoute{})
}

func (d *Dispatcher) Execute(ctx context.Context, line string) error {
	tokens, err := ParseCommandLine(line)
	if err != nil {
		return apperr.Invalid("Could not parse command line.", "Check quotes and escape characters.")
	}
	if len(tokens) == 0 {
		return nil
	}
	if tokens[0] == "pbdash" {
		tokens = tokens[1:]
		if len(tokens) == 0 {
			return nil
		}
	}

	switch tokens[0] {
	case "help":
		d.printHelp()
		return nil
	case "version":
		_, _ = fmt.Fprintln(d.stdout, d.version)
		return nil
	case "ui":
		return apperr.Invalid("Web UI is under development.", "")
	case "db":
		return d.execDB(argsAfterHead(tokens))
	case "superuser":
		return d.execSuperuser(argsAfterHead(tokens))
	case "context":
		return d.execContext(argsAfterHead(tokens))
	case "api":
		return d.execAPI(ctx, argsAfterHead(tokens))
	case "exit", "quit":
		return ErrExitRequested
	default:
		return apperr.Invalid("Unknown command `"+tokens[0]+"`.", "Run `help` to see available commands.")
	}
}

func argsAfterHead(tokens []string) []string {
	if len(tokens) <= 1 {
		return []string{}
	}
	return tokens[1:]
}

func (d *Dispatcher) printHelp() {
	help := strings.TrimSpace(`pbdash command reference

Run modes:
  pbdash                         Start full-screen TUI mode.
  pbdash -repl                   Start legacy REPL mode.
  pbdash -c "<command>"          Run one command and exit.
  pbdash <script-file>           Execute commands from a script file.
  pbdash -ui                     Reserved for the future web UI (currently under development).

Core commands:
  version                         Print CLI version.
  help                            Show available commands.

DB commands:
  db add --alias <dbAlias> --url <baseUrl>
                                  Save a PocketBase base URL as a db alias.
  db list                         List saved db aliases.
  db remove --alias <dbAlias>     Remove a saved db alias.

Superuser commands:
  superuser add --db <dbAlias> --alias <superuserAlias> --email <email> --password <password>
                                  Save superuser credentials for a db alias.
  superuser list --db <dbAlias>   List superuser aliases for a db alias.
  superuser remove --db <dbAlias> --alias <superuserAlias>
                                  Remove a saved superuser alias.

Context commands:
  context show                    Show current session/saved target context.
  context use --db <dbAlias> [--superuser <superuserAlias>]
                                  Set active session target context.
  context save                    Save current session context as default.
  context clear                   Clear current session context.
  context unsave                  Remove saved default context.

API commands (read-only GET):
  api collections --db <dbAlias> --superuser <superuserAlias> [--format table|csv|markdown] [--out <path>]
                                  List collections from PocketBase.
  api collection --db <dbAlias> --superuser <superuserAlias> --name <collectionName> [--format table|csv|markdown] [--out <path>]
                                  Get one collection by name.
  api records --db <dbAlias> --superuser <superuserAlias> --collection <collectionName> [--page <n>] [--per-page <n>] [--sort <expr>] [--filter <expr>] [--view auto|tui|table] [--format table|csv|markdown] [--out <path>]
                                  List records with paging, sort, and filter options.
  api record --db <dbAlias> --superuser <superuserAlias> --collection <collectionName> --id <recordId> [--format table|csv|markdown] [--out <path>]
                                  Get one record by id.

Output:
  Default format is table.
  csv/markdown requires --out <path>.
  TUI view requires a TTY terminal.`)
	_, _ = fmt.Fprintln(d.stdout, help)
}
