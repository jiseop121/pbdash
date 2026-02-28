package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"multi-pocketbase-ui/internal/apperr"
	"multi-pocketbase-ui/internal/pocketbase"
	"multi-pocketbase-ui/internal/storage"
)

type DispatcherConfig struct {
	Stdout  io.Writer
	Version string
	DataDir string
}

type Dispatcher struct {
	stdout   io.Writer
	version  string
	dbStore  *storage.DBStore
	suStore  *storage.SuperuserStore
	pbClient *pocketbase.Client
}

func NewDispatcher(cfg DispatcherConfig) *Dispatcher {
	return &Dispatcher{
		stdout:   cfg.Stdout,
		version:  cfg.Version,
		dbStore:  storage.NewDBStore(cfg.DataDir),
		suStore:  storage.NewSuperuserStore(cfg.DataDir),
		pbClient: pocketbase.NewClient(),
	}
}

func (d *Dispatcher) Execute(ctx context.Context, line string) error {
	tokens, err := ParseCommandLine(line)
	if err != nil {
		return apperr.Invalid("Could not parse command line.", "Check quotes and escape characters.")
	}
	if len(tokens) == 0 {
		return nil
	}
	if tokens[0] == "pbmulti" {
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
		return apperr.Invalid("UI mode is not available in Track 1.", "")
	case "db":
		return d.execDB(tokens[1:])
	case "superuser":
		return d.execSuperuser(tokens[1:])
	case "api":
		return d.execAPI(ctx, tokens[1:])
	case "exit", "quit":
		return ErrExitRequested
	default:
		return apperr.Invalid("Unknown command `"+tokens[0]+"`.", "Run `help` to see available commands.")
	}
}

func (d *Dispatcher) execDB(args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing db subcommand.", "Use: db add|list|remove")
	}
	cmd := args[0]
	switch cmd {
	case "add":
		fs := newFlagSet("db add")
		alias := fs.String("alias", "", "")
		baseURL := fs.String("url", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *alias == "" || *baseURL == "" {
			return apperr.Invalid("Missing required options `--alias` and `--url`.", "Example: db add --alias dev --url http://127.0.0.1:8090")
		}
		if err := d.dbStore.Add(*alias, *baseURL); err != nil {
			return mapStoreError(err)
		}
		_, _ = fmt.Fprintf(d.stdout, "Saved db alias %q.\n", *alias)
		return nil
	case "list":
		if len(args) > 1 {
			return apperr.Invalid("`db list` does not accept extra arguments.", "Use: db list")
		}
		items, err := d.dbStore.List()
		if err != nil {
			return mapStoreError(err)
		}
		rows := make([]map[string]any, 0, len(items))
		for _, it := range items {
			rows = append(rows, map[string]any{"db_alias": it.Alias, "base_url": it.BaseURL})
		}
		_, _ = fmt.Fprintln(d.stdout, renderTable([]string{"db_alias", "base_url"}, rows))
		return nil
	case "remove":
		fs := newFlagSet("db remove")
		alias := fs.String("alias", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *alias == "" {
			return apperr.Invalid("Missing required option `--alias`.", "Example: db remove --alias dev")
		}
		if err := d.dbStore.Remove(*alias); err != nil {
			return mapStoreError(err)
		}
		_, _ = fmt.Fprintf(d.stdout, "Removed db alias %q.\n", *alias)
		return nil
	default:
		return apperr.Invalid("Unknown db subcommand `"+cmd+"`.", "Use: db add|list|remove")
	}
}

func (d *Dispatcher) execSuperuser(args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing superuser subcommand.", "Use: superuser add|list|remove")
	}
	cmd := args[0]
	switch cmd {
	case "add":
		fs := newFlagSet("superuser add")
		dbAlias := fs.String("db", "", "")
		alias := fs.String("alias", "", "")
		email := fs.String("email", "", "")
		password := fs.String("password", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *dbAlias == "" || *alias == "" || *email == "" || *password == "" {
			return apperr.Invalid("Missing required options for superuser add.", "Example: superuser add --db dev --alias root --email admin@example.com --password secret")
		}
		if _, found, err := d.dbStore.Find(*dbAlias); err != nil {
			return mapStoreError(err)
		} else if !found {
			return apperr.Invalid("Could not find a saved db named \""+*dbAlias+"\".", "Run `pbmulti db list` to see available db aliases.")
		}
		if err := d.suStore.Add(*dbAlias, *alias, *email, *password); err != nil {
			return mapStoreError(err)
		}
		_, _ = fmt.Fprintf(d.stdout, "Saved superuser alias %q for db %q.\n", *alias, *dbAlias)
		return nil
	case "list":
		fs := newFlagSet("superuser list")
		dbAlias := fs.String("db", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *dbAlias == "" {
			return apperr.Invalid("Missing required option `--db`.", "Example: superuser list --db dev")
		}
		items, err := d.suStore.ListByDB(*dbAlias)
		if err != nil {
			return mapStoreError(err)
		}
		rows := make([]map[string]any, 0, len(items))
		for _, it := range items {
			rows = append(rows, map[string]any{"db_alias": it.DBAlias, "superuser_alias": it.Alias, "email": it.Email})
		}
		_, _ = fmt.Fprintln(d.stdout, renderTable([]string{"db_alias", "superuser_alias", "email"}, rows))
		return nil
	case "remove":
		fs := newFlagSet("superuser remove")
		dbAlias := fs.String("db", "", "")
		alias := fs.String("alias", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *dbAlias == "" || *alias == "" {
			return apperr.Invalid("Missing required options `--db` and `--alias`.", "Example: superuser remove --db dev --alias root")
		}
		if err := d.suStore.Remove(*dbAlias, *alias); err != nil {
			return mapStoreError(err)
		}
		_, _ = fmt.Fprintf(d.stdout, "Removed superuser alias %q from db %q.\n", *alias, *dbAlias)
		return nil
	default:
		return apperr.Invalid("Unknown superuser subcommand `"+cmd+"`.", "Use: superuser add|list|remove")
	}
}

func (d *Dispatcher) execAPI(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing api subcommand.", "Use: api collections|collection|records|record")
	}

	sub := args[0]
	switch sub {
	case "collections":
		fs := newFlagSet("api collections")
		dbAlias := fs.String("db", "", "")
		suAlias := fs.String("superuser", "", "")
		format := fs.String("format", "table", "")
		out := fs.String("out", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if err := validateOutputOptions(*format, *out); err != nil {
			return err
		}
		target, err := d.resolveTarget(*dbAlias, *suAlias)
		if err != nil {
			return err
		}
		token, err := d.authenticate(ctx, target)
		if err != nil {
			return err
		}
		payload, err := d.pbClient.GetJSON(ctx, target.DB.BaseURL, token, pocketbase.BuildCollectionsEndpoint(), nil)
		if err != nil {
			return mapPBError(err, target.SU.Alias, target.DB.Alias)
		}
		result := pocketbase.ParseItemsResult(payload)
		return d.writeQueryResult(*format, *out, result)

	case "collection":
		fs := newFlagSet("api collection")
		dbAlias := fs.String("db", "", "")
		suAlias := fs.String("superuser", "", "")
		name := fs.String("name", "", "")
		format := fs.String("format", "table", "")
		out := fs.String("out", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *name == "" {
			return apperr.Invalid("Missing required option `--name`.", "Example: api collection --name posts")
		}
		if err := validateOutputOptions(*format, *out); err != nil {
			return err
		}
		target, err := d.resolveTarget(*dbAlias, *suAlias)
		if err != nil {
			return err
		}
		token, err := d.authenticate(ctx, target)
		if err != nil {
			return err
		}
		payload, err := d.pbClient.GetJSON(ctx, target.DB.BaseURL, token, pocketbase.BuildCollectionEndpoint(*name), nil)
		if err != nil {
			return mapPBError(err, target.SU.Alias, target.DB.Alias)
		}
		result := pocketbase.ParseSingleResult(payload)
		return d.writeQueryResult(*format, *out, result)

	case "records":
		fs := newFlagSet("api records")
		dbAlias := fs.String("db", "", "")
		suAlias := fs.String("superuser", "", "")
		collection := fs.String("collection", "", "")
		page := fs.String("page", "", "")
		perPage := fs.String("per-page", "", "")
		sortExpr := fs.String("sort", "", "")
		filterExpr := fs.String("filter", "", "")
		format := fs.String("format", "table", "")
		out := fs.String("out", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *collection == "" {
			return apperr.Invalid("Missing required option `--collection`.", "Example: api records --collection posts")
		}
		if err := validateOutputOptions(*format, *out); err != nil {
			return err
		}
		query := map[string]string{}
		if *page != "" {
			if _, err := positiveInt(*page); err != nil {
				return apperr.Invalid("Invalid `--page` value.", "`--page` must be a positive integer.")
			}
			query["page"] = *page
		}
		if *perPage != "" {
			if _, err := positiveInt(*perPage); err != nil {
				return apperr.Invalid("Invalid `--per-page` value.", "`--per-page` must be a positive integer.")
			}
			query["perPage"] = *perPage
		}
		if *sortExpr != "" {
			query["sort"] = *sortExpr
		}
		if *filterExpr != "" {
			query["filter"] = *filterExpr
		}
		target, err := d.resolveTarget(*dbAlias, *suAlias)
		if err != nil {
			return err
		}
		token, err := d.authenticate(ctx, target)
		if err != nil {
			return err
		}
		payload, err := d.pbClient.GetJSON(ctx, target.DB.BaseURL, token, pocketbase.BuildRecordsEndpoint(*collection), query)
		if err != nil {
			return mapPBError(err, target.SU.Alias, target.DB.Alias)
		}
		result := pocketbase.ParseItemsResult(payload)
		return d.writeQueryResult(*format, *out, result)

	case "record":
		fs := newFlagSet("api record")
		dbAlias := fs.String("db", "", "")
		suAlias := fs.String("superuser", "", "")
		collection := fs.String("collection", "", "")
		recordID := fs.String("id", "", "")
		format := fs.String("format", "table", "")
		out := fs.String("out", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if *collection == "" || *recordID == "" {
			return apperr.Invalid("Missing required options `--collection` and `--id`.", "Example: api record --collection posts --id rec123")
		}
		if err := validateOutputOptions(*format, *out); err != nil {
			return err
		}
		target, err := d.resolveTarget(*dbAlias, *suAlias)
		if err != nil {
			return err
		}
		token, err := d.authenticate(ctx, target)
		if err != nil {
			return err
		}
		payload, err := d.pbClient.GetJSON(ctx, target.DB.BaseURL, token, pocketbase.BuildRecordEndpoint(*collection, *recordID), nil)
		if err != nil {
			return mapPBError(err, target.SU.Alias, target.DB.Alias)
		}
		result := pocketbase.ParseSingleResult(payload)
		return d.writeQueryResult(*format, *out, result)
	default:
		return apperr.Invalid("This CLI is read-only for PocketBase API operations.", "Only GET requests are supported.")
	}
}

type pbTarget struct {
	DB storage.DB
	SU storage.Superuser
}

func (d *Dispatcher) resolveTarget(dbAlias, suAlias string) (pbTarget, error) {
	if strings.TrimSpace(dbAlias) == "" || strings.TrimSpace(suAlias) == "" {
		return pbTarget{}, apperr.Invalid("Missing required options `--db` and `--superuser`.", "Example: --db dev --superuser root")
	}
	db, found, err := d.dbStore.Find(dbAlias)
	if err != nil {
		return pbTarget{}, mapStoreError(err)
	}
	if !found {
		return pbTarget{}, apperr.Invalid("Could not find a saved db named \""+dbAlias+"\".", "Run `pbmulti db list` to see available db aliases.")
	}
	su, found, err := d.suStore.Find(dbAlias, suAlias)
	if err != nil {
		return pbTarget{}, mapStoreError(err)
	}
	if !found {
		return pbTarget{}, apperr.Invalid("Superuser alias \""+suAlias+"\" is not configured for db \""+dbAlias+"\".", "Run `pbmulti superuser list --db "+dbAlias+"` to see available aliases.")
	}
	return pbTarget{DB: db, SU: su}, nil
}

func (d *Dispatcher) authenticate(ctx context.Context, target pbTarget) (string, error) {
	token, err := d.pbClient.Authenticate(ctx, target.DB.BaseURL, target.SU.Email, target.SU.Password)
	if err != nil {
		return "", mapPBError(err, target.SU.Alias, target.DB.Alias)
	}
	return token, nil
}

func (d *Dispatcher) writeQueryResult(format, out string, result pocketbase.QueryResult) error {
	text, err := RenderQueryResult(format, out, result)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(d.stdout, text)
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func invalidFlagError(err error) error {
	if err == nil {
		return nil
	}
	return apperr.Invalid("Invalid command arguments.", err.Error())
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	var validationErr *storage.ValidationError
	if errors.As(err, &validationErr) {
		return apperr.Invalid(validationErr.Message, "")
	}
	return apperr.RuntimeErr("Local configuration storage failed.", "Check local file permissions and retry.", err)
}

func mapPBError(err error, superuserAlias, dbAlias string) error {
	if err == nil {
		return nil
	}
	var authErr *pocketbase.AuthError
	if errors.As(err, &authErr) {
		return apperr.ExternalErr("Authentication failed for superuser \""+superuserAlias+"\" on db \""+dbAlias+"\".", "Verify the saved credentials for this superuser alias.", err)
	}
	if pocketbase.IsNetworkError(err) {
		return apperr.ExternalErr("Network request to PocketBase failed.", "Check db URL and network connectivity.", err)
	}
	var apiErr *pocketbase.APIError
	if errors.As(err, &apiErr) {
		return apperr.ExternalErr(fmt.Sprintf("PocketBase API request failed with status %d.", apiErr.Status), "Check credentials, query parameters, and target resource.", err)
	}
	return apperr.ExternalErr("PocketBase request failed.", "Check connectivity and server status.", err)
}

func positiveInt(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, fmt.Errorf("must be positive")
	}
	return v, nil
}

func validateOutputOptions(format, out string) error {
	normalized, err := pocketbase.ValidateFormat(format)
	if err != nil {
		return apperr.Invalid("Unsupported output format.", "Use one of: table, csv, markdown.")
	}
	if normalized == "table" && strings.TrimSpace(out) != "" {
		return apperr.Invalid("`--out` cannot be used when `--format` is `table`.", "Remove `--out` or switch to `csv`/`markdown`.")
	}
	if (normalized == "csv" || normalized == "markdown") && strings.TrimSpace(out) == "" {
		return apperr.Invalid("Missing required option `--out` when `--format` is `csv` or `markdown`.", "Example: --format csv --out ./records.csv")
	}
	return nil
}

func (d *Dispatcher) printHelp() {
	help := strings.TrimSpace(`pbmulti commands:
  version
  help
  db add --alias <dbAlias> --url <baseUrl>
  db list
  db remove --alias <dbAlias>
  superuser add --db <dbAlias> --alias <superuserAlias> --email <email> --password <password>
  superuser list --db <dbAlias>
  superuser remove --db <dbAlias> --alias <superuserAlias>
  api collections --db <dbAlias> --superuser <superuserAlias> [--format table|csv|markdown] [--out <path>]
  api collection --db <dbAlias> --superuser <superuserAlias> --name <collectionName> [--format table|csv|markdown] [--out <path>]
  api records --db <dbAlias> --superuser <superuserAlias> --collection <collectionName> [--page <n>] [--per-page <n>] [--sort <expr>] [--filter <expr>] [--format table|csv|markdown] [--out <path>]
  api record --db <dbAlias> --superuser <superuserAlias> --collection <collectionName> --id <recordId> [--format table|csv|markdown] [--out <path>]`)
	_, _ = fmt.Fprintln(d.stdout, help)
}
