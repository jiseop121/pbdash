package pocketbase

import (
	"fmt"
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
	return fmt.Sprintf("/api/collections/%s", name)
}

func BuildRecordsEndpoint(collection string) string {
	return fmt.Sprintf("/api/collections/%s/records", collection)
}

func BuildRecordEndpoint(collection, id string) string {
	return fmt.Sprintf("/api/collections/%s/records/%s", collection, id)
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
		rows := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		return rows
	}

	if records, ok := payload["records"].([]any); ok {
		rows := make([]map[string]any, 0, len(records))
		for _, item := range records {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		return rows
	}

	if len(payload) == 0 {
		return []map[string]any{}
	}

	return []map[string]any{payload}
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
		if cols[i] == "id" {
			return true
		}
		if cols[j] == "id" {
			return false
		}
		if cols[i] == "title" {
			return true
		}
		if cols[j] == "title" {
			return false
		}
		if cols[i] == "created" {
			return true
		}
		if cols[j] == "created" {
			return false
		}
		return cols[i] < cols[j]
	})
	return cols
}
