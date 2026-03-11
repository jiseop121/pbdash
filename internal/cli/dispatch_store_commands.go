package cli

import (
	"fmt"
	"strings"

	"github.com/jiseop121/pbdash/internal/apperr"
	"github.com/jiseop121/pbdash/internal/storage"
)

type localConfigSnapshot struct {
	dbs        []storage.DB
	superusers []storage.Superuser
	sessionCtx commandContext
	savedCtx   commandContext
	hasSaved   bool
	authCache  map[authCacheKey]authCacheEntry
}

func (d *Dispatcher) execDB(args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing db subcommand.", "Use: db add|list|remove")
	}

	switch args[0] {
	case "add":
		return d.execDBAdd(args[1:])
	case "list":
		return d.execDBList(args[1:])
	case "remove":
		return d.execDBRemove(args[1:])
	default:
		return apperr.Invalid("Unknown db subcommand `"+args[0]+"`.", "Use: db add|list|remove")
	}
}

func (d *Dispatcher) execSuperuser(args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing superuser subcommand.", "Use: superuser add|list|remove")
	}

	switch args[0] {
	case "add":
		return d.execSuperuserAdd(args[1:])
	case "list":
		return d.execSuperuserList(args[1:])
	case "remove":
		return d.execSuperuserRemove(args[1:])
	default:
		return apperr.Invalid("Unknown superuser subcommand `"+args[0]+"`.", "Use: superuser add|list|remove")
	}
}

func (d *Dispatcher) execContext(args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing context subcommand.", "Use: context show|use|save|clear|unsave")
	}

	switch args[0] {
	case "show":
		return d.execContextShow(args[1:])
	case "use":
		return d.execContextUse(args[1:])
	case "save":
		return d.execContextSave(args[1:])
	case "clear":
		return d.execContextClear(args[1:])
	case "unsave":
		return d.execContextUnsave(args[1:])
	default:
		return apperr.Invalid("Unknown context subcommand `"+args[0]+"`.", "Use: context show|use|save|clear|unsave")
	}
}

func (d *Dispatcher) execDBAdd(args []string) error {
	fs := newFlagSet("db add")
	alias := fs.String("alias", "", "")
	baseURL := fs.String("url", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *alias == "" || *baseURL == "" {
		return apperr.Invalid("Missing required options `--alias` and `--url`.", "Example: db add --alias dev --url http://127.0.0.1:8090")
	}

	db, err := d.saveDBAlias(*alias, *baseURL)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(d.stdout, "Saved db alias %q.\n", db.Alias)
	return nil
}

func (d *Dispatcher) execDBList(args []string) error {
	if len(args) > 0 {
		return apperr.Invalid("`db list` does not accept extra arguments.", "Use: db list")
	}

	items, err := d.dbStore.List()
	if err != nil {
		return mapStoreError(err)
	}
	_, _ = fmt.Fprintln(d.stdout, renderTable([]string{"db_alias", "base_url"}, dbRows(items)))
	return nil
}

func (d *Dispatcher) execDBRemove(args []string) error {
	fs := newFlagSet("db remove")
	alias := fs.String("alias", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *alias == "" {
		return apperr.Invalid("Missing required option `--alias`.", "Example: db remove --alias dev")
	}

	if err := d.removeDBAlias(*alias); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(d.stdout, "Removed db alias %q.\n", *alias)
	return nil
}

func (d *Dispatcher) execSuperuserAdd(args []string) error {
	fs := newFlagSet("superuser add")
	dbAlias := fs.String("db", "", "")
	alias := fs.String("alias", "", "")
	email := fs.String("email", "", "")
	password := fs.String("password", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *dbAlias == "" || *alias == "" || *email == "" || *password == "" {
		return apperr.Invalid("Missing required options for superuser add.", "Example: superuser add --db dev --alias root --email admin@example.com --password secret")
	}

	su, err := d.saveSuperuser(*dbAlias, *alias, *email, *password)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(d.stdout, "Saved superuser alias %q for db %q.\n", su.Alias, su.DBAlias)
	return nil
}

func (d *Dispatcher) execSuperuserList(args []string) error {
	fs := newFlagSet("superuser list")
	dbAlias := fs.String("db", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *dbAlias == "" {
		return apperr.Invalid("Missing required option `--db`.", "Example: superuser list --db dev")
	}

	items, err := d.suStore.ListByDB(*dbAlias)
	if err != nil {
		return mapStoreError(err)
	}
	_, _ = fmt.Fprintln(d.stdout, renderTable([]string{"db_alias", "superuser_alias", "email"}, superuserRows(items)))
	return nil
}

func (d *Dispatcher) execSuperuserRemove(args []string) error {
	fs := newFlagSet("superuser remove")
	dbAlias := fs.String("db", "", "")
	alias := fs.String("alias", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *dbAlias == "" || *alias == "" {
		return apperr.Invalid("Missing required options `--db` and `--alias`.", "Example: superuser remove --db dev --alias root")
	}

	if err := d.removeSuperuser(*dbAlias, *alias); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(d.stdout, "Removed superuser alias %q from db %q.\n", *alias, *dbAlias)
	return nil
}

func (d *Dispatcher) execContextShow(args []string) error {
	if len(args) > 0 {
		return apperr.Invalid("`context show` does not accept extra arguments.", "Use: context show")
	}

	rows := contextRows(d.sessionCtx, d.savedCtx, d.hasSaved)
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(d.stdout, "No context configured.")
		return nil
	}

	_, _ = fmt.Fprintln(d.stdout, renderTable([]string{"source", "db_alias", "superuser_alias"}, rows))
	return nil
}

func (d *Dispatcher) execContextUse(args []string) error {
	fs := newFlagSet("context use")
	dbAlias := fs.String("db", "", "")
	suAlias := fs.String("superuser", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if strings.TrimSpace(*dbAlias) == "" {
		return apperr.Invalid("Missing required option `--db`.", "Example: context use --db dev --superuser root")
	}

	next, err := d.buildContextSelection(*dbAlias, *suAlias)
	if err != nil {
		return err
	}
	d.sessionCtx = next
	_, _ = fmt.Fprintln(d.stdout, formatContextMessage("Updated session context", d.sessionCtx))
	return nil
}

func (d *Dispatcher) execContextSave(args []string) error {
	if len(args) > 0 {
		return apperr.Invalid("`context save` does not accept extra arguments.", "Use: context save")
	}
	if strings.TrimSpace(d.sessionCtx.DBAlias) == "" {
		return apperr.Invalid("No session context to save.", "Run `context use --db <alias> [--superuser <alias>]` first.")
	}

	if err := d.persistSavedContext(d.sessionCtx); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(d.stdout, formatContextMessage("Saved default context", d.sessionCtx))
	return nil
}

func (d *Dispatcher) execContextClear(args []string) error {
	if len(args) > 0 {
		return apperr.Invalid("`context clear` does not accept extra arguments.", "Use: context clear")
	}
	d.sessionCtx = commandContext{}
	_, _ = fmt.Fprintln(d.stdout, "Cleared session context.")
	return nil
}

func (d *Dispatcher) execContextUnsave(args []string) error {
	if len(args) > 0 {
		return apperr.Invalid("`context unsave` does not accept extra arguments.", "Use: context unsave")
	}
	if err := d.clearSavedContextState(); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(d.stdout, "Removed saved default context.")
	return nil
}

func (d *Dispatcher) saveDBAlias(alias, baseURL string) (storage.DB, error) {
	if err := d.dbStore.Add(alias, baseURL); err != nil {
		return storage.DB{}, mapStoreError(err)
	}
	return d.reloadDBAlias(alias, "Saved db alias could not be reloaded.")
}

func (d *Dispatcher) updateDBAlias(currentAlias, nextAlias, baseURL string) (storage.DB, error) {
	var updated storage.DB
	err := d.runWithLocalConfigRollback(func() error {
		if err := d.dbStore.Update(currentAlias, nextAlias, baseURL); err != nil {
			return mapStoreError(err)
		}
		if !strings.EqualFold(currentAlias, nextAlias) {
			if err := d.renameDBAliasReferences(currentAlias, nextAlias); err != nil {
				return err
			}
		}
		reloaded, err := d.reloadDBAlias(nextAlias, "Updated db alias could not be reloaded.")
		if err != nil {
			return err
		}
		updated = reloaded
		return nil
	})
	if err != nil {
		return storage.DB{}, err
	}
	return updated, nil
}

func (d *Dispatcher) removeDBAlias(alias string) error {
	return d.runWithLocalConfigRollback(func() error {
		if err := d.dbStore.Remove(alias); err != nil {
			return mapStoreError(err)
		}
		if err := d.suStore.RemoveByDB(alias); err != nil {
			return mapStoreError(err)
		}
		d.clearAuthCacheByDB(alias)
		return d.dropContextByDB(alias)
	})
}

func (d *Dispatcher) saveSuperuser(dbAlias, alias, email, password string) (storage.Superuser, error) {
	if err := d.requireDBAlias(dbAlias); err != nil {
		return storage.Superuser{}, err
	}
	if err := d.suStore.Add(dbAlias, alias, email, password); err != nil {
		return storage.Superuser{}, mapStoreError(err)
	}
	return d.reloadSuperuser(dbAlias, alias, "Saved superuser alias could not be reloaded.")
}

func (d *Dispatcher) updateSuperuser(dbAlias, currentAlias, nextAlias, email, password string) (storage.Superuser, error) {
	var updated storage.Superuser
	err := d.runWithLocalConfigRollback(func() error {
		if err := d.suStore.Update(dbAlias, currentAlias, nextAlias, email, password); err != nil {
			return mapStoreError(err)
		}
		if !strings.EqualFold(currentAlias, nextAlias) {
			if err := d.renameContextSuperuserAlias(dbAlias, currentAlias, nextAlias); err != nil {
				return err
			}
			d.clearAuthCacheForAlias(dbAlias, currentAlias)
		}
		reloaded, err := d.reloadSuperuser(dbAlias, nextAlias, "Updated superuser alias could not be reloaded.")
		if err != nil {
			return err
		}
		updated = reloaded
		return nil
	})
	if err != nil {
		return storage.Superuser{}, err
	}
	return updated, nil
}

func (d *Dispatcher) removeSuperuser(dbAlias, alias string) error {
	return d.runWithLocalConfigRollback(func() error {
		if err := d.suStore.Remove(dbAlias, alias); err != nil {
			return mapStoreError(err)
		}
		d.clearAuthCacheForAlias(dbAlias, alias)
		return d.dropContextBySuperuser(dbAlias, alias)
	})
}

func (d *Dispatcher) buildContextSelection(dbAlias, suAlias string) (commandContext, error) {
	db, found, err := d.dbStore.Find(dbAlias)
	if err != nil {
		return commandContext{}, mapStoreError(err)
	}
	if !found {
		return commandContext{}, apperr.Invalid("Could not find a saved db named \""+dbAlias+"\".", "Run `pbdash db list` to see available db aliases.")
	}

	next := commandContext{DBAlias: db.Alias, SuperuserAlias: strings.TrimSpace(suAlias)}
	if next.SuperuserAlias == "" {
		return next, nil
	}
	if _, found, err := d.suStore.Find(db.Alias, next.SuperuserAlias); err != nil {
		return commandContext{}, mapStoreError(err)
	} else if !found {
		return commandContext{}, apperr.Invalid("Superuser alias \""+next.SuperuserAlias+"\" is not configured for db \""+db.Alias+"\".", "Run `pbdash superuser list --db "+db.Alias+"` to see available aliases.")
	}
	return next, nil
}

func (d *Dispatcher) requireDBAlias(dbAlias string) error {
	if _, found, err := d.dbStore.Find(dbAlias); err != nil {
		return mapStoreError(err)
	} else if !found {
		return apperr.Invalid("Could not find a saved db named \""+dbAlias+"\".", "Run `pbdash db list` to see available db aliases.")
	}
	return nil
}

func (d *Dispatcher) reloadDBAlias(alias, message string) (storage.DB, error) {
	db, found, err := d.dbStore.Find(alias)
	if err != nil {
		return storage.DB{}, mapStoreError(err)
	}
	if !found {
		return storage.DB{}, apperr.RuntimeErr(message, "", nil)
	}
	return db, nil
}

func (d *Dispatcher) reloadSuperuser(dbAlias, alias, message string) (storage.Superuser, error) {
	su, found, err := d.suStore.Find(dbAlias, alias)
	if err != nil {
		return storage.Superuser{}, mapStoreError(err)
	}
	if !found {
		return storage.Superuser{}, apperr.RuntimeErr(message, "", nil)
	}
	return su, nil
}

func (d *Dispatcher) renameDBAliasReferences(currentAlias, nextAlias string) error {
	if err := d.suStore.ReassignDBAlias(currentAlias, nextAlias); err != nil {
		return mapStoreError(err)
	}
	if err := d.renameContextDBAlias(currentAlias, nextAlias); err != nil {
		return err
	}
	d.clearAuthCacheByDB(currentAlias)
	return nil
}

func (d *Dispatcher) persistSavedContext(ctx commandContext) error {
	err := d.saveSavedContext(storage.Context{DBAlias: ctx.DBAlias, SuperuserAlias: ctx.SuperuserAlias})
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
		if err := d.clearSavedContextState(); err != nil {
			return err
		}
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

func (d *Dispatcher) renameContextDBAlias(currentAlias, nextAlias string) error {
	if strings.EqualFold(d.sessionCtx.DBAlias, currentAlias) {
		d.sessionCtx.DBAlias = nextAlias
	}
	if d.hasSaved && strings.EqualFold(d.savedCtx.DBAlias, currentAlias) {
		d.savedCtx.DBAlias = nextAlias
		if err := d.persistSavedContext(d.savedCtx); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) renameContextSuperuserAlias(dbAlias, currentAlias, nextAlias string) error {
	if strings.EqualFold(d.sessionCtx.DBAlias, dbAlias) && strings.EqualFold(d.sessionCtx.SuperuserAlias, currentAlias) {
		d.sessionCtx.SuperuserAlias = nextAlias
	}
	if d.hasSaved && strings.EqualFold(d.savedCtx.DBAlias, dbAlias) && strings.EqualFold(d.savedCtx.SuperuserAlias, currentAlias) {
		d.savedCtx.SuperuserAlias = nextAlias
		if err := d.persistSavedContext(d.savedCtx); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) clearAuthCacheByDB(dbAlias string) {
	for key := range d.authCache {
		if strings.EqualFold(key.dbAlias, dbAlias) {
			delete(d.authCache, key)
		}
	}
}

func (d *Dispatcher) clearAuthCacheForAlias(dbAlias, suAlias string) {
	delete(d.authCache, authCacheKey{dbAlias: strings.ToLower(dbAlias), suAlias: strings.ToLower(suAlias)})
}

func (d *Dispatcher) runWithLocalConfigRollback(run func() error) error {
	snapshot, err := d.snapshotLocalConfig()
	if err != nil {
		return err
	}
	if err := run(); err != nil {
		if restoreErr := d.restoreLocalConfig(snapshot); restoreErr != nil {
			return apperr.RuntimeErr("Could not rollback local config after a failed update.", "", fmt.Errorf("%v; rollback failed: %w", err, restoreErr))
		}
		return err
	}
	return nil
}

func (d *Dispatcher) snapshotLocalConfig() (localConfigSnapshot, error) {
	dbs, err := d.dbStore.List()
	if err != nil {
		return localConfigSnapshot{}, mapStoreError(err)
	}
	superusers, err := d.suStore.List()
	if err != nil {
		return localConfigSnapshot{}, mapStoreError(err)
	}
	return localConfigSnapshot{
		dbs:        dbs,
		superusers: superusers,
		sessionCtx: d.sessionCtx,
		savedCtx:   d.savedCtx,
		hasSaved:   d.hasSaved,
		authCache:  cloneAuthCache(d.authCache),
	}, nil
}

func (d *Dispatcher) restoreLocalConfig(snapshot localConfigSnapshot) error {
	if err := d.dbStore.ReplaceAll(snapshot.dbs); err != nil {
		return mapStoreError(err)
	}
	if err := d.suStore.ReplaceAll(snapshot.superusers); err != nil {
		return mapStoreError(err)
	}
	d.sessionCtx = snapshot.sessionCtx
	d.savedCtx = snapshot.savedCtx
	d.hasSaved = snapshot.hasSaved
	d.authCache = cloneAuthCache(snapshot.authCache)
	if snapshot.hasSaved {
		if err := d.saveSavedContext(storage.Context{
			DBAlias:        snapshot.savedCtx.DBAlias,
			SuperuserAlias: snapshot.savedCtx.SuperuserAlias,
		}); err != nil {
			return mapStoreError(err)
		}
		return nil
	}
	if err := d.clearSavedContextState(); err != nil {
		return err
	}
	d.sessionCtx = snapshot.sessionCtx
	d.authCache = cloneAuthCache(snapshot.authCache)
	return nil
}

func (d *Dispatcher) clearSavedContextState() error {
	if err := d.clearSavedContext(); err != nil {
		return mapStoreError(err)
	}
	d.savedCtx = commandContext{}
	d.hasSaved = false
	return nil
}

func cloneAuthCache(src map[authCacheKey]authCacheEntry) map[authCacheKey]authCacheEntry {
	cloned := make(map[authCacheKey]authCacheEntry, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func dbRows(items []storage.DB) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"db_alias": item.Alias,
			"base_url": item.BaseURL,
		})
	}
	return rows
}

func superuserRows(items []storage.Superuser) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"db_alias":        item.DBAlias,
			"superuser_alias": item.Alias,
			"email":           item.Email,
		})
	}
	return rows
}

func contextRows(sessionCtx, savedCtx commandContext, hasSaved bool) []map[string]any {
	rows := []map[string]any{}
	if strings.TrimSpace(sessionCtx.DBAlias) != "" || strings.TrimSpace(sessionCtx.SuperuserAlias) != "" {
		rows = append(rows, map[string]any{
			"source":          "session",
			"db_alias":        sessionCtx.DBAlias,
			"superuser_alias": sessionCtx.SuperuserAlias,
		})
	}
	if hasSaved {
		rows = append(rows, map[string]any{
			"source":          "saved",
			"db_alias":        savedCtx.DBAlias,
			"superuser_alias": savedCtx.SuperuserAlias,
		})
	}
	return rows
}

func formatContextMessage(prefix string, ctx commandContext) string {
	if strings.TrimSpace(ctx.SuperuserAlias) == "" {
		return fmt.Sprintf("%s: db=%q.", prefix, ctx.DBAlias)
	}
	return fmt.Sprintf("%s: db=%q superuser=%q.", prefix, ctx.DBAlias, ctx.SuperuserAlias)
}
