package cli

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"multi-pocketbase-ui/internal/pocketbase"
)

func TestRecordsTUIHandleKeySupportsArrowHorizontalNavigation(t *testing.T) {
	ui := &recordsTUI{
		tableView: tview.NewTable(),
		result:    pocketbase.QueryResult{Rows: []map[string]any{rowWithColumns(visibleColumnWindow + 2)}},
	}

	ui.handleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if ui.columnOffset != 1 {
		t.Fatalf("column offset mismatch after right: got=%d want=1", ui.columnOffset)
	}

	ui.handleKey(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if ui.columnOffset != 0 {
		t.Fatalf("column offset mismatch after left: got=%d want=0", ui.columnOffset)
	}
}

func TestRecordsTUIArrowHorizontalNavigationBounds(t *testing.T) {
	ui := &recordsTUI{
		tableView: tview.NewTable(),
		result:    pocketbase.QueryResult{Rows: []map[string]any{rowWithColumns(visibleColumnWindow + 1)}},
	}

	ui.handleKey(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if ui.columnOffset != 0 {
		t.Fatalf("left at start should stay 0, got=%d", ui.columnOffset)
	}

	for i := 0; i < visibleColumnWindow+4; i++ {
		ui.handleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	}
	if ui.columnOffset != 1 {
		t.Fatalf("right bound mismatch: got=%d want=1", ui.columnOffset)
	}
}

func rowWithColumns(count int) map[string]any {
	row := make(map[string]any, count)
	for i := 0; i < count; i++ {
		row[fmt.Sprintf("c%02d", i)] = i
	}
	return row
}
