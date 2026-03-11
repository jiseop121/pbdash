package pocketbase

import (
	"strings"
	"testing"
)

func TestBuildEndpointsEscapeDynamicSegments(t *testing.T) {
	collectionEndpoint := BuildCollectionEndpoint("a/b")
	if strings.Contains(collectionEndpoint, "/a/b") {
		t.Fatalf("collection endpoint should escape slash: %s", collectionEndpoint)
	}
	if !strings.Contains(collectionEndpoint, "a%2Fb") {
		t.Fatalf("collection endpoint missing escaped segment: %s", collectionEndpoint)
	}

	recordsEndpoint := BuildRecordsEndpoint("x y")
	if !strings.Contains(recordsEndpoint, "x%20y") {
		t.Fatalf("records endpoint missing escaped segment: %s", recordsEndpoint)
	}

	recordEndpoint := BuildRecordEndpoint("../posts", "rec/1")
	if strings.Contains(recordEndpoint, "../") || strings.Contains(recordEndpoint, "/rec/1") {
		t.Fatalf("record endpoint should escape traversal and slash: %s", recordEndpoint)
	}
	if !strings.Contains(recordEndpoint, "..%2Fposts") {
		t.Fatalf("record endpoint should contain escaped traversal: %s", recordEndpoint)
	}
	if !strings.Contains(recordEndpoint, "rec%2F1") {
		t.Fatalf("record endpoint should contain escaped id: %s", recordEndpoint)
	}
}

func TestCollectColumnsPriorityOrdering(t *testing.T) {
	rows := []map[string]any{{
		"z":       "v",
		"created": "2026-01-01",
		"title":   "hello",
		"id":      "1",
		"a":       "b",
	}}

	cols := CollectColumns(rows)
	want := []string{"id", "title", "created", "a", "z"}
	if len(cols) != len(want) {
		t.Fatalf("column count mismatch: got=%d want=%d (%v)", len(cols), len(want), cols)
	}
	for i := range want {
		if cols[i] != want[i] {
			t.Fatalf("column order mismatch at %d: got=%q want=%q (%v)", i, cols[i], want[i], cols)
		}
	}
}

func TestParseItemsResultSupportsRecordsField(t *testing.T) {
	result := ParseItemsResult(map[string]any{
		"records": []any{
			map[string]any{"id": "1", "title": "hello"},
			"skip-me",
			map[string]any{"id": "2"},
		},
		"page":       float64(2),
		"perPage":    float64(50),
		"totalItems": float64(2),
		"totalPages": float64(1),
	})

	if len(result.Rows) != 2 {
		t.Fatalf("row count mismatch: got=%d want=2 (%v)", len(result.Rows), result.Rows)
	}
	wantIDs := map[string]struct{}{"1": {}, "2": {}}
	for _, row := range result.Rows {
		id, ok := row["id"].(string)
		if !ok {
			t.Fatalf("each extracted row should keep its string id: %v", result.Rows)
		}
		if _, ok := wantIDs[id]; !ok {
			t.Fatalf("unexpected row extracted from records payload: %v", result.Rows)
		}
		delete(wantIDs, id)
	}
	if len(wantIDs) != 0 {
		t.Fatalf("non-record items should be skipped; missing ids=%v rows=%v", wantIDs, result.Rows)
	}
	if result.Meta == nil || result.Meta.Page != 2 || result.Meta.PerPage != 50 || result.Meta.TotalItems != 2 || result.Meta.TotalPages != 1 {
		t.Fatalf("unexpected metadata: %+v", result.Meta)
	}
}
