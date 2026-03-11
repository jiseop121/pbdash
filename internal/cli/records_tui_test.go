package cli

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jiseop121/pbdash/internal/pocketbase"
	"github.com/jiseop121/pbdash/internal/storage"
)

func TestNavigatorTUIHandleKeySupportsArrowHorizontalNavigation(t *testing.T) {
	ui := &navigatorTUI{
		screen:    screenRecords,
		tableView: tview.NewTable(),
		result:    queryResultWithColumns(visibleColumnWindow + 2),
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

func TestNavigatorTUIArrowHorizontalNavigationBounds(t *testing.T) {
	ui := &navigatorTUI{
		screen:    screenRecords,
		tableView: tview.NewTable(),
		result:    queryResultWithColumns(visibleColumnWindow + 1),
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

func TestPickPreferredSuperuser(t *testing.T) {
	candidates := []storage.Superuser{
		{DBAlias: "dev", Alias: "ops", Email: "ops@example.com"},
		{DBAlias: "dev", Alias: "root", Email: "root@example.com"},
	}

	got, ok := pickPreferredSuperuser("dev", candidates, commandContext{DBAlias: "dev", SuperuserAlias: "root"}, commandContext{})
	if !ok || got.Alias != "root" {
		t.Fatalf("expected session context superuser, got=%+v ok=%v", got, ok)
	}

	got, ok = pickPreferredSuperuser("dev", candidates, commandContext{}, commandContext{DBAlias: "dev", SuperuserAlias: "ops"})
	if !ok || got.Alias != "ops" {
		t.Fatalf("expected saved context superuser, got=%+v ok=%v", got, ok)
	}

	got, ok = pickPreferredSuperuser("dev", candidates[:1], commandContext{}, commandContext{})
	if !ok || got.Alias != "ops" {
		t.Fatalf("expected single configured superuser, got=%+v ok=%v", got, ok)
	}
}

func TestRunRecordsTUIUsesNavigatorRunner(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	target := pbTarget{
		DB: storage.DB{Alias: "dev", BaseURL: "http://127.0.0.1:8090"},
		SU: storage.Superuser{DBAlias: "dev", Alias: "root", Email: "root@example.com"},
	}
	state := RecordsQueryState{Collection: "posts", Page: 2}

	var gotRoute navigatorRoute
	d.navigatorRunner = func(_ context.Context, route navigatorRoute) error {
		gotRoute = route
		return nil
	}

	if err := d.runRecordsTUI(context.Background(), target, state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotRoute.hasTarget || gotRoute.screen != screenRecords {
		t.Fatalf("unexpected route: %+v", gotRoute)
	}
	if gotRoute.target.DB.Alias != "dev" || gotRoute.target.SU.Alias != "root" {
		t.Fatalf("target mismatch: %+v", gotRoute.target)
	}
	if gotRoute.state.Collection != "posts" || gotRoute.state.Page != 2 {
		t.Fatalf("state mismatch: %+v", gotRoute.state)
	}
}

func TestNavigatorTUISetupViewsCapturesTableShortcuts(t *testing.T) {
	ui := newTestNavigatorTUI()

	ui.setupViews()
	handler := ui.tableView.InputHandler()
	require.NotNil(t, handler)
	assert.Same(t, ui.tableView, ui.app.GetFocus())

	handler(tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone), func(tview.Primitive) {})
	assert.Equal(t, 1, ui.selectedIndex)

	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	assert.False(t, ui.detailVisible)
}

func TestNavigatorTUIRenderCurrentScreenRestoresMainFocus(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.setupViews()

	ui.app.SetFocus(ui.detailView)
	ui.renderCurrentScreen()

	assert.Same(t, ui.tableView, ui.app.GetFocus())
}

func TestNavigatorTUIPushAndBackRestoreMainFocus(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.setupViews()

	ui.app.SetFocus(ui.detailView)
	ui.pushScreen(screenCollections)
	assert.Same(t, ui.tableView, ui.app.GetFocus())

	ui.app.SetFocus(ui.detailView)
	ui.goBack()
	assert.Same(t, ui.tableView, ui.app.GetFocus())
}

func TestNavigatorTUICloseAndDismissErrorRestoreMainFocus(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.setupViews()

	ui.modalOpen = true
	ui.pages.AddPage("test-modal", tview.NewTextView(), true, true)
	ui.closeModal("test-modal")
	assert.False(t, ui.modalOpen)
	assert.False(t, ui.pages.HasPage("test-modal"))
	assert.Same(t, ui.tableView, ui.app.GetFocus())

	ui.showError(assert.AnError)
	require.True(t, ui.modalOpen)
	require.True(t, ui.pages.HasPage("error"))

	ui.dismissErrorModal()
	assert.False(t, ui.modalOpen)
	assert.False(t, ui.pages.HasPage("error"))
	assert.Same(t, ui.tableView, ui.app.GetFocus())
}

func TestNavigatorTUIRefreshShortcutRestoresMainFocus(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	_, err := dispatcher.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)

	ui := newTestNavigatorTUI()
	ui.dispatcher = dispatcher
	ui.screen = screenDBList
	ui.dbs = []storage.DB{{Alias: "stale", BaseURL: "http://stale"}}
	ui.setupViews()

	ui.app.SetFocus(ui.detailView)
	handler := ui.tableView.InputHandler()
	require.NotNil(t, handler)

	handler(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone), func(tview.Primitive) {})

	require.Len(t, ui.dbs, 1)
	assert.Equal(t, "dev", ui.dbs[0].Alias)
	assert.Same(t, ui.tableView, ui.app.GetFocus())
}

func TestNavigatorTUIHandleKeyManagerShortcutsAndQuit(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	_, err := dispatcher.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)

	ui := newTestNavigatorTUI()
	ui.dispatcher = dispatcher
	ui.screen = screenDBList
	ui.dbs = []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}}

	stopped := false
	ui.stop = func() {
		stopped = true
	}

	ui.setupViews()
	handler := ui.tableView.InputHandler()
	require.NotNil(t, handler)

	handler(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone), func(tview.Primitive) {})
	require.True(t, ui.modalOpen)
	assert.True(t, ui.pages.HasPage("db-manager"))
	ui.closeModal("db-manager")

	handler(tcell.NewEventKey(tcell.KeyRune, 'u', tcell.ModNone), func(tview.Primitive) {})
	require.True(t, ui.modalOpen)
	assert.True(t, ui.pages.HasPage("superuser-manager"))
	ui.closeModal("superuser-manager")

	handler(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone), func(tview.Primitive) {})
	assert.True(t, stopped)
}

func TestDBManagerStateSelectAlias(t *testing.T) {
	manager := newDBManagerState([]storage.DB{
		{Alias: "dev", BaseURL: "http://127.0.0.1:8090"},
	})

	db, ok := manager.selectAlias("dev")
	require.True(t, ok)
	assert.Equal(t, "dev", manager.selectedAlias)
	assert.Equal(t, "http://127.0.0.1:8090", db.BaseURL)

	_, ok = manager.selectAlias(managerNewOption)
	require.False(t, ok)
	assert.Empty(t, manager.selectedAlias)
}

func TestDBManagerStateSaveAndRemove(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})

	manager := dbManagerState{}
	require.NoError(t, manager.save(dispatcher, "dev", "http://127.0.0.1:8090"))

	saved, found, err := dispatcher.dbStore.Find("dev")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "http://127.0.0.1:8090", saved.BaseURL)

	manager.selectedAlias = "dev"
	require.NoError(t, manager.save(dispatcher, "local", "http://127.0.0.1:8091"))

	_, found, err = dispatcher.dbStore.Find("dev")
	require.NoError(t, err)
	assert.False(t, found)

	updated, found, err := dispatcher.dbStore.Find("local")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "http://127.0.0.1:8091", updated.BaseURL)

	manager.selectedAlias = "local"
	require.NoError(t, manager.remove(dispatcher))

	_, found, err = dispatcher.dbStore.Find("local")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestDBManagerStateRemoveRequiresSelection(t *testing.T) {
	err := (dbManagerState{}).remove(NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Select an existing db alias first")
}

func TestSuperuserManagerStateSelectAlias(t *testing.T) {
	manager := superuserManagerState{
		selectedDB: "dev",
		superusers: []storage.Superuser{
			{DBAlias: "dev", Alias: "root", Email: "root@example.com"},
		},
	}

	su, ok := manager.selectAlias("root")
	require.True(t, ok)
	assert.Equal(t, "root", manager.selectedAlias)
	assert.Equal(t, "root@example.com", su.Email)

	_, ok = manager.selectAlias(managerNewOption)
	require.False(t, ok)
	assert.Empty(t, manager.selectedAlias)
}

func TestNewSuperuserManagerStateFallsBackToFirstDB(t *testing.T) {
	manager := newSuperuserManagerState([]storage.DB{
		{Alias: "dev", BaseURL: "http://127.0.0.1:8090"},
		{Alias: "prod", BaseURL: "https://pb.example.com"},
	}, "")

	assert.Equal(t, "dev", manager.selectedDB)
	assert.Equal(t, 0, manager.selectedDBIndex())
}

func TestNavigatorTUIRetargetAliasesAfterRename(t *testing.T) {
	ui := &navigatorTUI{
		hasTarget: true,
		target: pbTarget{
			DB: storage.DB{Alias: "dev", BaseURL: "http://127.0.0.1:8090"},
			SU: storage.Superuser{DBAlias: "dev", Alias: "root", Email: "root@example.com"},
		},
	}

	ui.retargetDBAlias("dev", "prod")
	assert.Equal(t, "prod", ui.target.DB.Alias)
	assert.Equal(t, "prod", ui.target.SU.DBAlias)

	ui.retargetSuperuserAlias("prod", "root", "admin")
	assert.Equal(t, "admin", ui.target.SU.Alias)
}

func TestSuperuserManagerStateSaveAndRemove(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	_, err := dispatcher.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)

	manager := superuserManagerState{selectedDB: "dev"}
	require.NoError(t, manager.save(dispatcher, "root", "root@example.com", "secret"))

	saved, found, err := dispatcher.suStore.Find("dev", "root")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "root@example.com", saved.Email)

	manager.selectedAlias = "root"
	require.NoError(t, manager.save(dispatcher, "ops", "ops@example.com", ""))

	_, found, err = dispatcher.suStore.Find("dev", "root")
	require.NoError(t, err)
	assert.False(t, found)

	updated, found, err := dispatcher.suStore.Find("dev", "ops")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "ops@example.com", updated.Email)

	manager.selectedAlias = "ops"
	require.NoError(t, manager.remove(dispatcher))

	_, found, err = dispatcher.suStore.Find("dev", "ops")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestSuperuserManagerStateRemoveRequiresSelection(t *testing.T) {
	err := (superuserManagerState{selectedDB: "dev"}).remove(NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Select an existing superuser first")
}

func newTestNavigatorTUI() *navigatorTUI {
	return &navigatorTUI{
		app:           tview.NewApplication(),
		stop:          func() {},
		statusView:    tview.NewTextView(),
		tableView:     tview.NewTable(),
		detailView:    tview.NewTextView(),
		helpView:      tview.NewTextView(),
		screen:        screenRecords,
		detailVisible: true,
		observedCols:  map[string]struct{}{},
		result: pocketbase.QueryResult{Rows: []map[string]any{
			{"id": "1", "title": "first"},
			{"id": "2", "title": "second"},
		}},
	}
}

func queryResultWithColumns(count int) pocketbaseQueryResult {
	rows := []map[string]any{rowWithColumns(count)}
	return pocketbase.QueryResult{Rows: rows}
}

type pocketbaseQueryResult = pocketbase.QueryResult

func rowWithColumns(count int) map[string]any {
	row := make(map[string]any, count)
	for i := 0; i < count; i++ {
		row[fmt.Sprintf("c%02d", i)] = i
	}
	return row
}
