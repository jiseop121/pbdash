package cli

import (
	"fmt"
	"strings"

	"multi-pocketbase-ui/internal/apperr"
	"multi-pocketbase-ui/internal/storage"
)

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
		if err := d.dropContextByDB(*alias); err != nil {
			return err
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
			return apperr.Invalid("Could not find a saved db named \""+*dbAlias+"\".", "Run `pbviewer db list` to see available db aliases.")
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
		if err := d.dropContextBySuperuser(*dbAlias, *alias); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(d.stdout, "Removed superuser alias %q from db %q.\n", *alias, *dbAlias)
		return nil
	default:
		return apperr.Invalid("Unknown superuser subcommand `"+cmd+"`.", "Use: superuser add|list|remove")
	}
}

func (d *Dispatcher) execContext(args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing context subcommand.", "Use: context show|use|save|clear|unsave")
	}

	sub := args[0]
	switch sub {
	case "show":
		if len(args) > 1 {
			return apperr.Invalid("`context show` does not accept extra arguments.", "Use: context show")
		}
		rows := []map[string]any{}
		if strings.TrimSpace(d.sessionCtx.DBAlias) != "" || strings.TrimSpace(d.sessionCtx.SuperuserAlias) != "" {
			rows = append(rows, map[string]any{
				"source":          "session",
				"db_alias":        d.sessionCtx.DBAlias,
				"superuser_alias": d.sessionCtx.SuperuserAlias,
			})
		}
		if d.hasSaved {
			rows = append(rows, map[string]any{
				"source":          "saved",
				"db_alias":        d.savedCtx.DBAlias,
				"superuser_alias": d.savedCtx.SuperuserAlias,
			})
		}
		if len(rows) == 0 {
			_, _ = fmt.Fprintln(d.stdout, "No context configured.")
			return nil
		}
		_, _ = fmt.Fprintln(d.stdout, renderTable([]string{"source", "db_alias", "superuser_alias"}, rows))
		return nil

	case "use":
		fs := newFlagSet("context use")
		dbAlias := fs.String("db", "", "")
		suAlias := fs.String("superuser", "", "")
		if err := fs.Parse(args[1:]); err != nil {
			return invalidFlagError(err)
		}
		if strings.TrimSpace(*dbAlias) == "" {
			return apperr.Invalid("Missing required option `--db`.", "Example: context use --db dev --superuser root")
		}
		db, found, err := d.dbStore.Find(*dbAlias)
		if err != nil {
			return mapStoreError(err)
		}
		if !found {
			return apperr.Invalid("Could not find a saved db named \""+*dbAlias+"\".", "Run `pbviewer db list` to see available db aliases.")
		}
		if strings.TrimSpace(*suAlias) != "" {
			if _, found, err := d.suStore.Find(db.Alias, *suAlias); err != nil {
				return mapStoreError(err)
			} else if !found {
				return apperr.Invalid("Superuser alias \""+*suAlias+"\" is not configured for db \""+db.Alias+"\".", "Run `pbviewer superuser list --db "+db.Alias+"` to see available aliases.")
			}
		}
		d.sessionCtx = commandContext{DBAlias: db.Alias, SuperuserAlias: strings.TrimSpace(*suAlias)}
		if d.sessionCtx.SuperuserAlias == "" {
			_, _ = fmt.Fprintf(d.stdout, "Updated session context: db=%q.\n", d.sessionCtx.DBAlias)
			return nil
		}
		_, _ = fmt.Fprintf(d.stdout, "Updated session context: db=%q superuser=%q.\n", d.sessionCtx.DBAlias, d.sessionCtx.SuperuserAlias)
		return nil

	case "save":
		if len(args) > 1 {
			return apperr.Invalid("`context save` does not accept extra arguments.", "Use: context save")
		}
		if strings.TrimSpace(d.sessionCtx.DBAlias) == "" {
			return apperr.Invalid("No session context to save.", "Run `context use --db <alias> [--superuser <alias>]` first.")
		}
		if err := d.persistSavedContext(d.sessionCtx); err != nil {
			return err
		}
		if d.sessionCtx.SuperuserAlias == "" {
			_, _ = fmt.Fprintf(d.stdout, "Saved default context: db=%q.\n", d.sessionCtx.DBAlias)
			return nil
		}
		_, _ = fmt.Fprintf(d.stdout, "Saved default context: db=%q superuser=%q.\n", d.sessionCtx.DBAlias, d.sessionCtx.SuperuserAlias)
		return nil

	case "clear":
		if len(args) > 1 {
			return apperr.Invalid("`context clear` does not accept extra arguments.", "Use: context clear")
		}
		d.sessionCtx = commandContext{}
		_, _ = fmt.Fprintln(d.stdout, "Cleared session context.")
		return nil

	case "unsave":
		if len(args) > 1 {
			return apperr.Invalid("`context unsave` does not accept extra arguments.", "Use: context unsave")
		}
		if err := d.ctxStore.Clear(); err != nil {
			return mapStoreError(err)
		}
		d.savedCtx = commandContext{}
		d.hasSaved = false
		_, _ = fmt.Fprintln(d.stdout, "Removed saved default context.")
		return nil

	default:
		return apperr.Invalid("Unknown context subcommand `"+sub+"`.", "Use: context show|use|save|clear|unsave")
	}
}

func (d *Dispatcher) persistSavedContext(ctx commandContext) error {
	err := d.ctxStore.Save(storage.Context{DBAlias: ctx.DBAlias, SuperuserAlias: ctx.SuperuserAlias})
	if err != nil {
		return mapStoreError(err)
	}
	d.savedCtx = ctx
	d.hasSaved = true
	return nil
}

func (d *Dispatcher) dropContextByDB(dbAlias string) error {
	if strings.EqualFold(d.sessionCtx.DBAlias, dbAlias) {
		d.sessionCtx = commandContext{}
	}
	if d.hasSaved && strings.EqualFold(d.savedCtx.DBAlias, dbAlias) {
		if err := d.ctxStore.Clear(); err != nil {
			return mapStoreError(err)
		}
		d.savedCtx = commandContext{}
		d.hasSaved = false
	}
	return nil
}

func (d *Dispatcher) dropContextBySuperuser(dbAlias, suAlias string) error {
	if strings.EqualFold(d.sessionCtx.DBAlias, dbAlias) && strings.EqualFold(d.sessionCtx.SuperuserAlias, suAlias) {
		d.sessionCtx.SuperuserAlias = ""
	}
	if d.hasSaved && strings.EqualFold(d.savedCtx.DBAlias, dbAlias) && strings.EqualFold(d.savedCtx.SuperuserAlias, suAlias) {
		d.savedCtx.SuperuserAlias = ""
		if err := d.persistSavedContext(d.savedCtx); err != nil {
			return err
		}
	}
	return nil
}
