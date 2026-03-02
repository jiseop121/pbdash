package cli

import (
	"sort"
	"strings"
)

var topLevelCommands = []string{"help", "version", "db", "superuser", "context", "api", "exit", "quit"}

var subcommands = map[string][]string{
	"db":        {"add", "list", "remove"},
	"superuser": {"add", "list", "remove"},
	"context":   {"show", "use", "save", "clear", "unsave"},
	"api":       {"collections", "collection", "records", "record"},
}

var commandFlags = map[string][]string{
	"db add":           {"--alias", "--url"},
	"db remove":        {"--alias"},
	"superuser add":    {"--db", "--alias", "--email", "--password"},
	"superuser list":   {"--db"},
	"superuser remove": {"--db", "--alias"},
	"context use":      {"--db", "--superuser"},
	"api collections":  {"--db", "--superuser", "--format", "--out"},
	"api collection":   {"--db", "--superuser", "--name", "--format", "--out"},
	"api records":      {"--db", "--superuser", "--collection", "--page", "--per-page", "--sort", "--filter", "--view", "--format", "--out"},
	"api record":       {"--db", "--superuser", "--collection", "--id", "--format", "--out"},
}

func (d *Dispatcher) Complete(line string) []string {
	tokens, current := splitCompletionTokens(line)

	if len(tokens) == 0 {
		return filterPrefix(topLevelCommands, current)
	}

	if len(tokens) == 1 {
		if _, ok := subcommands[tokens[0]]; ok {
			return filterPrefix(subcommands[tokens[0]], current)
		}
		return filterPrefix(topLevelCommands, current)
	}

	prev := tokens[len(tokens)-1]
	if strings.HasPrefix(prev, "--") {
		return filterPrefix(d.flagValueSuggestions(tokens, prev), current)
	}

	key := commandKey(tokens)
	if strings.HasPrefix(current, "--") || current == "" {
		if flags, ok := commandFlags[key]; ok {
			return filterPrefix(flags, current)
		}
	}

	if len(tokens) == 1 {
		if subs, ok := subcommands[tokens[0]]; ok {
			return filterPrefix(subs, current)
		}
	}

	return filterPrefix(d.flagValueSuggestions(tokens, ""), current)
}

func splitCompletionTokens(line string) ([]string, string) {
	trimmedRight := strings.TrimRight(line, " \t\n\r")
	if trimmedRight == "" {
		return nil, ""
	}
	hasTrailingSpace := len(trimmedRight) != len(line)
	fields := strings.Fields(trimmedRight)
	if hasTrailingSpace {
		return fields, ""
	}
	if len(fields) == 0 {
		return nil, ""
	}
	return fields[:len(fields)-1], fields[len(fields)-1]
}

func commandKey(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) == 1 {
		return tokens[0]
	}
	return tokens[0] + " " + tokens[1]
}

func (d *Dispatcher) flagValueSuggestions(tokens []string, activeFlag string) []string {
	if activeFlag == "" && len(tokens) > 0 {
		last := tokens[len(tokens)-1]
		if strings.HasPrefix(last, "--") {
			activeFlag = last
		}
	}

	switch activeFlag {
	case "--format":
		return []string{"table", "csv", "markdown"}
	case "--view":
		return []string{"auto", "tui", "table"}
	case "--db":
		return d.dbAliases()
	case "--superuser":
		dbAlias := optionValue(tokens, "--db")
		if dbAlias == "" {
			dbAlias = d.sessionCtx.DBAlias
		}
		if dbAlias == "" && d.hasSaved {
			dbAlias = d.savedCtx.DBAlias
		}
		return d.superuserAliases(dbAlias)
	default:
		return nil
	}
}

func optionValue(tokens []string, flag string) string {
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i] == flag {
			return tokens[i+1]
		}
	}
	return ""
}

func (d *Dispatcher) dbAliases() []string {
	items, err := d.dbStore.List()
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Alias)
	}
	return out
}

func (d *Dispatcher) superuserAliases(dbAlias string) []string {
	if strings.TrimSpace(dbAlias) != "" {
		items, err := d.suStore.ListByDB(dbAlias)
		if err != nil {
			return nil
		}
		out := make([]string, 0, len(items))
		for _, it := range items {
			out = append(out, it.Alias)
		}
		return out
	}

	all, err := d.suStore.List()
	if err != nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, it := range all {
		set[it.Alias] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for alias := range set {
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

func filterPrefix(candidates []string, prefix string) []string {
	if len(candidates) == 0 {
		return nil
	}
	lowerPrefix := strings.ToLower(prefix)
	out := make([]string, 0, len(candidates))
	for _, item := range candidates {
		if strings.HasPrefix(strings.ToLower(item), lowerPrefix) {
			out = append(out, item)
		}
	}
	return out
}
