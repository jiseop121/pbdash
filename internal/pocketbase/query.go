package pocketbase

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type PageMeta struct {
	Page       int
	PerPage    int
	TotalItems int
	TotalPages int
}

type QueryResult struct {
	Rows []map[string]any
	Meta *PageMeta
}

func ParseItemsResult(payload map[string]any) QueryResult {
	rows := extractRows(payload)
	return QueryResult{Rows: rows, Meta: extractMeta(payload)}
}

func ParseSingleResult(payload map[string]any) QueryResult {
	if payload == nil {
		return QueryResult{Rows: []map[string]any{}}
	}
	return QueryResult{Rows: []map[string]any{payload}}
}

func BuildCollectionsEndpoint() string {
	return "/api/collections"
}

func BuildCollectionEndpoint(name string) string {
	return fmt.Sprintf("/api/collections/%s", url.PathEscape(name))
}

func BuildRecordsEndpoint(collection string) string {
	return fmt.Sprintf("/api/collections/%s/records", url.PathEscape(collection))
}

func BuildRecordEndpoint(collection, id string) string {
	return fmt.Sprintf("/api/collections/%s/records/%s", url.PathEscape(collection), url.PathEscape(id))
}

func ValidateFormat(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return "table", nil
	}
	format = strings.ToLower(format)
	switch format {
	case "table", "csv", "markdown":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
}

func extractRows(payload map[string]any) []map[string]any {
	if payload == nil {
		return []map[string]any{}
	}

	if items, ok := payload["items"].([]any); ok {
		return extractRowsList(items)
	}

	if records, ok := payload["records"].([]any); ok {
		return extractRowsList(records)
	}

	if len(payload) == 0 {
		return []map[string]any{}
	}

	return []map[string]any{payload}
}

func extractRowsList(items []any) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func extractMeta(payload map[string]any) *PageMeta {
	if payload == nil {
		return nil
	}
	meta := &PageMeta{
		Page:       intFromAny(payload["page"]),
		PerPage:    intFromAny(payload["perPage"]),
		TotalItems: intFromAny(payload["totalItems"]),
		TotalPages: intFromAny(payload["totalPages"]),
	}
	if meta.Page == 0 && meta.PerPage == 0 && meta.TotalItems == 0 && meta.TotalPages == 0 {
		return nil
	}
	return meta
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}

func CollectColumns(rows []map[string]any) []string {
	keys := map[string]struct{}{}
	for _, row := range rows {
		for k := range row {
			keys[k] = struct{}{}
		}
	}
	cols := make([]string, 0, len(keys))
	for k := range keys {
		cols = append(cols, k)
	}
	sort.Slice(cols, func(i, j int) bool {
		if cols[i] == cols[j] {
			return false
		}
		pi := columnPriority(cols[i])
		pj := columnPriority(cols[j])
		if pi != pj {
			return pi < pj
		}
		return cols[i] < cols[j]
	})
	return cols
}

func columnPriority(col string) int {
	switch col {
	case "id":
		return 0
	case "title":
		return 1
	case "created":
		return 2
	default:
		return 3
	}
}
