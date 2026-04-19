package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiseop121/pbdash/internal/apperr"
	"github.com/jiseop121/pbdash/internal/pocketbase"
	"github.com/jiseop121/pbdash/internal/storage"
)

func (d *Dispatcher) execAPI(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return apperr.Invalid("Missing api subcommand.", "Use: api collections|collection|records|record")
	}

	switch args[0] {
	case "collections":
		return d.execAPICollections(ctx, args[1:])
	case "collection":
		return d.execAPICollection(ctx, args[1:])
	case "records":
		return d.execAPIRecords(ctx, args[1:])
	case "record":
		return d.execAPIRecord(ctx, args[1:])
	default:
		return apperr.Invalid("This CLI is read-only for PocketBase API operations.", "Only GET requests are supported.")
	}
}

func (d *Dispatcher) execAPICollections(ctx context.Context, args []string) error {
	fs := newFlagSet("api collections")
	dbAlias := fs.String("db", "", "")
	suAlias := fs.String("superuser", "", "")
	format := fs.String("format", "table", "")
	out := fs.String("out", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if _, err := validateOutputOptions(*format, *out); err != nil {
		return err
	}
	target, err := d.resolveSession(*dbAlias, *suAlias)
	if err != nil {
		return err
	}
	result, err := d.fetchCollections(ctx, target)
	if err != nil {
		return err
	}
	return d.writeQueryResult(*format, *out, result)
}

func (d *Dispatcher) execAPICollection(ctx context.Context, args []string) error {
	fs := newFlagSet("api collection")
	dbAlias := fs.String("db", "", "")
	suAlias := fs.String("superuser", "", "")
	name := fs.String("name", "", "")
	format := fs.String("format", "table", "")
	out := fs.String("out", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *name == "" {
		return apperr.Invalid("Missing required option `--name`.", "Example: api collection --name posts")
	}
	if _, err := validateOutputOptions(*format, *out); err != nil {
		return err
	}
	target, err := d.resolveSession(*dbAlias, *suAlias)
	if err != nil {
		return err
	}
	payload, err := d.getJSONWithAuth(ctx, target, pocketbase.BuildCollectionEndpoint(*name), nil)
	if err != nil {
		return err
	}
	return d.writeQueryResult(*format, *out, pocketbase.ParseSingleResult(payload))
}

func (d *Dispatcher) execAPIRecords(ctx context.Context, args []string) error {
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
	view := fs.String("view", "auto", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *collection == "" {
		return apperr.Invalid("Missing required option `--collection`.", "Example: api records --collection posts")
	}

	normalizedFormat, err := validateOutputOptions(*format, *out)
	if err != nil {
		return err
	}
	normalizedView, err := normalizeView(*view)
	if err != nil {
		return err
	}
	shouldTUI, err := shouldUseRecordsTUI(normalizedView, normalizedFormat, d.HasTTY())
	if err != nil {
		return err
	}

	state := RecordsQueryState{
		Collection: *collection,
		Sort:       *sortExpr,
		Filter:     *filterExpr,
	}
	if *page != "" {
		v, err := positiveInt(*page)
		if err != nil {
			return apperr.Invalid("Invalid `--page` value.", "`--page` must be a positive integer.")
		}
		state.Page = v
	}
	if *perPage != "" {
		v, err := positiveInt(*perPage)
		if err != nil {
			return apperr.Invalid("Invalid `--per-page` value.", "`--per-page` must be a positive integer.")
		}
		state.PerPage = v
	}

	target, err := d.resolveSession(*dbAlias, *suAlias)
	if err != nil {
		return err
	}
	if shouldTUI {
		return d.runRecordsTUI(ctx, target, state)
	}
	result, err := d.fetchRecords(ctx, target, state)
	if err != nil {
		return err
	}
	return d.writeQueryResult(normalizedFormat, *out, result)
}

func (d *Dispatcher) execAPIRecord(ctx context.Context, args []string) error {
	fs := newFlagSet("api record")
	dbAlias := fs.String("db", "", "")
	suAlias := fs.String("superuser", "", "")
	collection := fs.String("collection", "", "")
	recordID := fs.String("id", "", "")
	format := fs.String("format", "table", "")
	out := fs.String("out", "", "")
	if err := fs.Parse(args); err != nil {
		return invalidFlagError(err)
	}
	if *collection == "" || *recordID == "" {
		return apperr.Invalid("Missing required options `--collection` and `--id`.", "Example: api record --collection posts --id rec123")
	}
	normalizedFormat, err := validateOutputOptions(*format, *out)
	if err != nil {
		return err
	}
	target, err := d.resolveSession(*dbAlias, *suAlias)
	if err != nil {
		return err
	}
	payload, err := d.getJSONWithAuth(ctx, target, pocketbase.BuildRecordEndpoint(*collection, *recordID), nil)
	if err != nil {
		return err
	}
	return d.writeQueryResult(normalizedFormat, *out, pocketbase.ParseSingleResult(payload))
}

func normalizeView(view string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(view))
	if v == "" {
		return "auto", nil
	}
	switch v {
	case "auto", "tui", "table":
		return v, nil
	default:
		return "", apperr.Invalid("Unsupported view mode.", "Use one of: auto, tui, table.")
	}
}

func shouldUseRecordsTUI(view, format string, hasTTY bool) (bool, error) {
	if view == "tui" && format != "table" {
		return false, apperr.Invalid("`--view tui` requires `--format table`.", "Use `--format table` or switch view to `table`/`auto`.")
	}

	switch view {
	case "tui":
		if !hasTTY {
			return false, apperr.Invalid("`--view tui` requires a TTY terminal.", "Run `pbdash` in a terminal and execute this command there.")
		}
		return true, nil
	case "auto":
		return format == "table" && hasTTY, nil
	case "table":
		return false, nil
	default:
		return false, apperr.Invalid("Unsupported view mode.", "Use one of: auto, tui, table.")
	}
}

type pbSession struct {
	DB storage.DB
	SU storage.Superuser
}

func (d *Dispatcher) resolveSession(dbAlias, suAlias string) (pbSession, error) {
	resolvedDB, resolvedSU, err := d.resolveAliases(dbAlias, suAlias)
	if err != nil {
		return pbSession{}, err
	}

	db, found, err := d.dbStore.Find(resolvedDB)
	if err != nil {
		return pbSession{}, mapStoreError(err)
	}
	if !found {
		return pbSession{}, apperr.Invalid("Could not find a saved db named \""+resolvedDB+"\".", "Run `pbdash db list` to see available db aliases.")
	}
	su, found, err := d.suStore.Find(db.Alias, resolvedSU)
	if err != nil {
		return pbSession{}, mapStoreError(err)
	}
	if !found {
		return pbSession{}, apperr.Invalid("Superuser alias \""+resolvedSU+"\" is not configured for db \""+db.Alias+"\".", "Run `pbdash superuser list --db "+db.Alias+"` to see available aliases.")
	}
	return pbSession{DB: db, SU: su}, nil
}

func (d *Dispatcher) resolveAliases(dbAlias, suAlias string) (string, string, error) {
	explicit := commandContext{DBAlias: strings.TrimSpace(dbAlias), SuperuserAlias: strings.TrimSpace(suAlias)}

	resolvedDB := explicit.DBAlias
	if resolvedDB == "" {
		if strings.TrimSpace(d.sessionCtx.DBAlias) != "" {
			resolvedDB = d.sessionCtx.DBAlias
		} else if d.hasSaved {
			resolvedDB = d.savedCtx.DBAlias
		}
	}

	resolvedSU := explicit.SuperuserAlias
	if resolvedSU == "" {
		if strings.TrimSpace(d.sessionCtx.SuperuserAlias) != "" && contextMatchesDB(d.sessionCtx, resolvedDB) {
			resolvedSU = d.sessionCtx.SuperuserAlias
		} else if d.hasSaved && strings.TrimSpace(d.savedCtx.SuperuserAlias) != "" && contextMatchesDB(d.savedCtx, resolvedDB) {
			resolvedSU = d.savedCtx.SuperuserAlias
		}
	}

	if strings.TrimSpace(resolvedDB) == "" || strings.TrimSpace(resolvedSU) == "" {
		return "", "", apperr.Invalid("Missing required options `--db` and `--superuser`.", "Set context with `context use --db <alias> --superuser <alias>` or provide flags explicitly.")
	}
	return resolvedDB, resolvedSU, nil
}

func contextMatchesDB(ctx commandContext, resolvedDB string) bool {
	if strings.TrimSpace(ctx.DBAlias) == "" {
		return true
	}
	if strings.TrimSpace(resolvedDB) == "" {
		return false
	}
	return strings.EqualFold(ctx.DBAlias, resolvedDB)
}

func (d *Dispatcher) fetchCollections(ctx context.Context, target pbSession) (pocketbase.QueryResult, error) {
	payload, err := d.getJSONWithAuth(ctx, target, pocketbase.BuildCollectionsEndpoint(), nil)
	if err != nil {
		return pocketbase.QueryResult{}, err
	}
	return pocketbase.ParseItemsResult(payload), nil
}

func (d *Dispatcher) fetchRecords(ctx context.Context, target pbSession, state RecordsQueryState) (pocketbase.QueryResult, error) {
	payload, err := d.getJSONWithAuth(ctx, target, pocketbase.BuildRecordsEndpoint(state.Collection), state.QueryParams())
	if err != nil {
		return pocketbase.QueryResult{}, err
	}
	return pocketbase.ParseItemsResult(payload), nil
}

func (d *Dispatcher) getJSONWithAuth(ctx context.Context, target pbSession, endpoint string, query map[string]string) (map[string]any, error) {
	token, err := d.authenticate(ctx, target, false)
	if err != nil {
		return nil, err
	}

	payload, err := d.pbClient.GetJSON(ctx, target.DB.BaseURL, token, endpoint, query)
	if err == nil {
		return payload, nil
	}
	var authErr *pocketbase.AuthError
	if !errors.As(err, &authErr) {
		return nil, mapPBError(err, target.SU.Alias, target.DB.Alias)
	}

	d.clearAuthCache(target)
	token, err = d.authenticate(ctx, target, true)
	if err != nil {
		return nil, err
	}
	payload, err = d.pbClient.GetJSON(ctx, target.DB.BaseURL, token, endpoint, query)
	if err != nil {
		return nil, mapPBError(err, target.SU.Alias, target.DB.Alias)
	}
	return payload, nil
}

func (d *Dispatcher) authenticate(ctx context.Context, target pbSession, force bool) (string, error) {
	if !force {
		if token, ok := d.getCachedToken(target); ok {
			return token, nil
		}
	}
	token, err := d.pbClient.Authenticate(ctx, target.DB.BaseURL, target.SU.Email, target.SU.Password)
	if err != nil {
		return "", mapPBError(err, target.SU.Alias, target.DB.Alias)
	}
	d.storeCachedToken(target, token)
	return token, nil
}

func (d *Dispatcher) getCachedToken(target pbSession) (string, bool) {
	entry, ok := d.authCache[authCacheKey{dbAlias: strings.ToLower(target.DB.Alias), suAlias: strings.ToLower(target.SU.Alias)}]
	if !ok {
		return "", false
	}
	if entry.hasExpiry {
		now := d.now().UTC()
		if !entry.expiresAt.After(now.Add(30 * time.Second)) {
			d.clearAuthCache(target)
			return "", false
		}
	}
	return entry.token, true
}

func (d *Dispatcher) storeCachedToken(target pbSession, token string) {
	entry := authCacheEntry{token: token}
	if expiresAt, ok := parseTokenExpiry(token); ok {
		entry.expiresAt = expiresAt
		entry.hasExpiry = true
	}
	key := authCacheKey{dbAlias: strings.ToLower(target.DB.Alias), suAlias: strings.ToLower(target.SU.Alias)}
	d.authCache[key] = entry
}

func (d *Dispatcher) clearAuthCache(target pbSession) {
	delete(d.authCache, authCacheKey{dbAlias: strings.ToLower(target.DB.Alias), suAlias: strings.ToLower(target.SU.Alias)})
}

func (d *Dispatcher) writeQueryResult(format, out string, result pocketbase.QueryResult) error {
	text, err := RenderQueryResult(format, out, result)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(d.stdout, text)
	return nil
}
