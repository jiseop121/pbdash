package cli

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jiseop121/pbdash/internal/pocketbase"
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

func TestRecordsTUISetupViewsCapturesTableShortcuts(t *testing.T) {
	ui := &recordsTUI{
		app:           tview.NewApplication(),
		statusView:    tview.NewTextView(),
		tableView:     tview.NewTable(),
		detailView:    tview.NewTextView(),
		helpView:      tview.NewTextView(),
		detailVisible: true,
		observedCols:  map[string]struct{}{},
		result: pocketbase.QueryResult{Rows: []map[string]any{
			{"id": "1", "title": "first"},
			{"id": "2", "title": "second"},
		}},
	}

	ui.setupViews()
	handler := ui.tableView.InputHandler()
	require.NotNil(t, handler)

	handler(tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone), func(tview.Primitive) {})
	assert.Equal(t, 1, ui.selectedIndex)

	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	assert.False(t, ui.detailVisible)
}

func rowWithColumns(count int) map[string]any {
	row := make(map[string]any, count)
	for i := 0; i < count; i++ {
		row[fmt.Sprintf("c%02d", i)] = i
	}
	return row
}
