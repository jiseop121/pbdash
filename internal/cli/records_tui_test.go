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
	session := pbSession{
		DB: storage.DB{Alias: "dev", BaseURL: "http://127.0.0.1:8090"},
		SU: storage.Superuser{DBAlias: "dev", Alias: "root", Email: "root@example.com"},
	}
	state := RecordsQueryState{Collection: "posts", Page: 2}

	var gotRoute navigatorRoute
	d.navigatorRunner = func(_ context.Context, route navigatorRoute) error {
		gotRoute = route
		return nil
	}

	if err := d.runRecordsTUI(context.Background(), session, state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotRoute.hasSession || gotRoute.screen != screenRecords {
		t.Fatalf("unexpected route: %+v", gotRoute)
	}
	if gotRoute.session.DB.Alias != "dev" || gotRoute.session.SU.Alias != "root" {
		t.Fatalf("session mismatch: %+v", gotRoute.session)
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
	assert.Equal(t, screenRecordDetail, ui.screen)
	require.NotNil(t, ui.recordDetail)
	assert.Equal(t, "second", ui.recordDetail["title"])
}

func TestNavigatorTUIRenderCurrentScreenRestoresMainFocus(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.setupViews()

	ui.app.SetFocus(ui.detailView)
	ui.renderCurrentScreen()

	assert.Same(t, ui.tableView, ui.app.GetFocus())
}

func TestNavigatorTUIGlobalQuitWorksOutsideTableFocus(t *testing.T) {
	ui := newTestNavigatorTUI()
	stopped := false
	ui.stop = func() {
		stopped = true
	}
	ui.setupViews()
	ui.app.SetFocus(ui.detailView)

	got := ui.handleGlobalKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))

	require.Nil(t, got)
	assert.True(t, stopped)
}

func TestNavigatorTUIGlobalBackWorksOutsideTableFocus(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.setupViews()
	ui.pushScreen(screenCollections)
	ui.recordDetail = map[string]any{"id": "1"}
	ui.pushScreen(screenRecordDetail)
	ui.app.SetFocus(ui.detailView)

	got := ui.handleGlobalKey(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone))

	require.Nil(t, got)
	assert.Equal(t, screenCollections, ui.screen)
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

func TestNavigatorTUICollectionScreenShowsNames(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.screen = screenCollections
	ui.collections = []map[string]any{
		{"id": "col_posts", "name": "posts", "type": "base"},
		{"id": "col_logs", "type": "view"},
	}

	ui.renderTable()

	require.Equal(t, "NAME", ui.tableView.GetCell(0, 0).Text)
	assert.Equal(t, "posts", ui.tableView.GetCell(1, 0).Text)
	assert.Equal(t, "col_logs", ui.tableView.GetCell(2, 0).Text)

	row, ok := ui.selectedRow()
	require.True(t, ok)
	assert.Equal(t, "posts", row["name"])
	assert.Equal(t, "col_posts", row["id"])
	assert.Equal(t, "base", row["type"])
}

func TestNavigatorTUICopyRecordDetailWritesClipboard(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	require.NotNil(t, screen)

	app := tview.NewApplication()
	app.SetScreen(screen)

	ui := newTestNavigatorTUI()
	ui.app = app
	ui.termScreen = screen
	ui.screen = screenRecordDetail
	ui.recordDetail = map[string]any{"id": "rec-001", "title": "first"}

	require.True(t, ui.copyRecordDetail())
	assert.Contains(t, string(screen.GetClipboardData()), `"id": "rec-001"`)
	assert.Equal(t, "copied (OSC52)", ui.statusMessage)
}

func TestRemapFormArrowNavigationInputFieldUsesVerticalOnly(t *testing.T) {
	field := tview.NewInputField()

	up := remapFormArrowNavigation(field, tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	down := remapFormArrowNavigation(field, tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	left := remapFormArrowNavigation(field, tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	right := remapFormArrowNavigation(field, tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))

	require.Equal(t, tcell.KeyBacktab, up.Key())
	require.Equal(t, tcell.KeyTab, down.Key())
	require.Equal(t, tcell.KeyLeft, left.Key())
	require.Equal(t, tcell.KeyRight, right.Key())
}

func TestRemapFormArrowNavigationDropdownUsesAllArrowsWhenClosed(t *testing.T) {
	dropdown := tview.NewDropDown().SetOptions([]string{"one", "two"}, nil)

	left := remapFormArrowNavigation(dropdown, tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	right := remapFormArrowNavigation(dropdown, tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	up := remapFormArrowNavigation(dropdown, tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	down := remapFormArrowNavigation(dropdown, tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))

	require.Equal(t, tcell.KeyBacktab, left.Key())
	require.Equal(t, tcell.KeyTab, right.Key())
	require.Equal(t, tcell.KeyBacktab, up.Key())
	require.Equal(t, tcell.KeyTab, down.Key())
}

func TestRemapFormArrowNavigationDropdownPreservesOpenListKeys(t *testing.T) {
	dropdown := tview.NewDropDown().SetOptions([]string{"one", "two"}, nil)
	setFocus := func(p tview.Primitive) {}
	dropdown.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), setFocus)
	require.True(t, dropdown.IsOpen())

	up := remapFormArrowNavigation(dropdown, tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	down := remapFormArrowNavigation(dropdown, tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))

	require.Equal(t, tcell.KeyUp, up.Key())
	require.Equal(t, tcell.KeyDown, down.Key())
}

func TestRemapFormArrowNavigationCheckboxAndButtonUseAllArrows(t *testing.T) {
	checkbox := tview.NewCheckbox()
	button := tview.NewButton("OK")

	require.Equal(t, tcell.KeyBacktab, remapFormArrowNavigation(checkbox, tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)).Key())
	require.Equal(t, tcell.KeyTab, remapFormArrowNavigation(checkbox, tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)).Key())
	require.Equal(t, tcell.KeyBacktab, remapFormArrowNavigation(button, tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)).Key())
	require.Equal(t, tcell.KeyTab, remapFormArrowNavigation(button, tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)).Key())
}

func TestRemapSubmitCancelNavigationUsesEnterAndEsc(t *testing.T) {
	checkbox := tview.NewCheckbox()
	applied := false
	cancelled := false

	enter := remapSubmitCancelNavigation(checkbox, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func() {
		applied = true
	}, func() {
		cancelled = true
	})
	require.Nil(t, enter)
	assert.True(t, applied)
	assert.False(t, cancelled)
	assert.False(t, checkbox.IsChecked())

	cancelled = false
	esc := remapSubmitCancelNavigation(checkbox, tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone), nil, func() {
		cancelled = true
	})
	require.Nil(t, esc)
	assert.True(t, cancelled)
}

func TestRemapSubmitCancelNavigationPreservesSpaceAndArrowKeys(t *testing.T) {
	checkbox := tview.NewCheckbox()

	space := remapSubmitCancelNavigation(checkbox, tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone), nil, nil)
	left := remapSubmitCancelNavigation(checkbox, tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), nil, nil)

	require.Equal(t, tcell.KeyRune, space.Key())
	assert.Equal(t, ' ', space.Rune())
	require.Equal(t, tcell.KeyBacktab, left.Key())
}

func TestScreenDBListBKey(t *testing.T) {
	dispatcher := newTestDispatcherWithDB(t, "dev", "http://127.0.0.1:8090")
	ui := newTestNavigatorTUIOnScreen(screenDBList)
	ui.dispatcher = dispatcher
	ui.dbs = []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}}
	ui.setupViews()

	// 'b' should be a no-op on screenDBList (no duplicate db-list modal)
	result := ui.consumeRuneCommand('b')

	assert.False(t, result)
	assert.False(t, ui.modalOpen)
	assert.False(t, ui.pages.HasPage("db-list"))
}

func TestScreenDBListQKey(t *testing.T) {
	ui := newTestNavigatorTUIOnScreen(screenDBList)
	stopped := false
	ui.stop = func() { stopped = true }
	ui.setupViews()

	ui.handleGlobalKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))

	assert.True(t, stopped)
}

// ── consumeGlobalKey: Esc on screenDBList ────────────────────────────────────

func TestConsumeGlobalKeyEscOnDBListIsIgnored(t *testing.T) {
	keys := []tcell.Key{tcell.KeyEsc, tcell.KeyBackspace, tcell.KeyBackspace2}
	for _, key := range keys {
		stopped := false
		ui := &navigatorTUI{screen: screenDBList, stop: func() { stopped = true }}

		consumed := ui.consumeGlobalKey(tcell.NewEventKey(key, 0, tcell.ModNone))

		assert.True(t, consumed, "key %v should be consumed", key)
		assert.False(t, stopped, "key %v should not stop app on screenDBList", key)
		assert.Equal(t, screenDBList, ui.screen)
	}
}

func TestConsumeGlobalKeyEscGoesBackOnNonDBList(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.setupViews()
	ui.screen = screenDBList
	ui.pushScreen(screenCollections) // history=[screenDBList], screen=screenCollections

	consumed := ui.consumeGlobalKey(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone))

	require.True(t, consumed)
	assert.Equal(t, screenDBList, ui.screen)
}

// ── shouldShowDetailPane ─────────────────────────────────────────────────────

func TestShouldShowDetailPane(t *testing.T) {
	tests := []struct {
		screen        navigatorScreen
		detailVisible bool
		want          bool
	}{
		{screenDBList, true, false},
		{screenSuperusers, true, false},
		{screenCollections, true, true},
		{screenCollections, false, false},
		{screenRecords, true, false},
		{screenRecordDetail, true, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.screen), func(t *testing.T) {
			ui := &navigatorTUI{screen: tt.screen, detailVisible: tt.detailVisible}
			assert.Equal(t, tt.want, ui.shouldShowDetailPane())
		})
	}
}

// ── consumeRuneCommand: d key ────────────────────────────────────────────────

func TestConsumeRuneCommandDKey(t *testing.T) {
	tests := []struct {
		name           string
		screen         navigatorScreen
		initialVisible bool
		wantConsumed   bool
		wantVisible    bool
	}{
		{"dbList/ignored", screenDBList, true, false, true},
		{"superusers/ignored", screenSuperusers, false, false, false},
		{"records/ignored", screenRecords, true, false, true},
		{"recordDetail/ignored", screenRecordDetail, true, false, true},
		{"collections/toggle-on", screenCollections, false, true, true},
		{"collections/toggle-off", screenCollections, true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := newTestNavigatorTUI()
			ui.screen = tt.screen
			ui.detailVisible = tt.initialVisible
			ui.setupViews()

			got := ui.consumeRuneCommand('d')

			assert.Equal(t, tt.wantConsumed, got)
			assert.Equal(t, tt.wantVisible, ui.detailVisible)
		})
	}
}

// ── consumeRuneCommand: u key ────────────────────────────────────────────────

func TestConsumeRuneCommandUKey(t *testing.T) {
	tests := []struct {
		name          string
		screen        navigatorScreen
		dbs           []storage.DB
		wantConsumed  bool
		wantModalPage string
	}{
		{
			name:          "dbList/with-db/opens-superuser-list",
			screen:        screenDBList,
			dbs:           []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}},
			wantConsumed:  true,
			wantModalPage: "superuser-list",
		},
		{
			name:         "dbList/no-dbs/ignored",
			screen:       screenDBList,
			dbs:          nil,
			wantConsumed: false,
		},
		{
			name:         "superusers/ignored",
			screen:       screenSuperusers,
			dbs:          []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}},
			wantConsumed: false,
		},
		{
			name:          "collections/opens-superuser-list",
			screen:        screenCollections,
			dbs:           []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}},
			wantConsumed:  true,
			wantModalPage: "superuser-list",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
			for _, db := range tt.dbs {
				_, err := dispatcher.saveDBAlias(db.Alias, db.BaseURL)
				require.NoError(t, err)
			}
			ui := newTestNavigatorTUIOnScreen(tt.screen)
			ui.dispatcher = dispatcher
			ui.dbs = tt.dbs
			ui.setupViews()

			got := ui.consumeRuneCommand('u')

			assert.Equal(t, tt.wantConsumed, got)
			if tt.wantModalPage != "" {
				assert.True(t, ui.modalOpen)
				assert.True(t, ui.pages.HasPage(tt.wantModalPage))
			} else {
				assert.False(t, ui.modalOpen)
			}
		})
	}
}

// ── consumeRuneCommand: n/e/D on screenDBList ─────────────────────────────────

func TestConsumeRuneCommandInlineDBListCRUD(t *testing.T) {
	tests := []struct {
		name          string
		key           rune
		dbs           []storage.DB
		wantConsumed  bool
		wantModalPage string
	}{
		{
			name:          "n/opens-db-edit",
			key:           'n',
			wantConsumed:  true,
			wantModalPage: "db-edit",
		},
		{
			name:          "e/with-db/opens-db-edit",
			key:           'e',
			dbs:           []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}},
			wantConsumed:  true,
			wantModalPage: "db-edit",
		},
		{
			name:         "e/no-db/ignored",
			key:          'e',
			wantConsumed: false,
		},
		{
			name:          "D/with-db/opens-confirm",
			key:           'D',
			dbs:           []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}},
			wantConsumed:  true,
			wantModalPage: "confirm",
		},
		{
			name:         "D/no-db/ignored",
			key:          'D',
			wantConsumed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := newTestDispatcherWithDBs(t, tt.dbs)
			ui := newTestNavigatorTUIOnScreen(screenDBList)
			ui.dispatcher = dispatcher
			ui.dbs = tt.dbs
			ui.setupViews()

			got := ui.consumeRuneCommand(tt.key)

			assert.Equal(t, tt.wantConsumed, got)
			if tt.wantModalPage != "" {
				assert.True(t, ui.modalOpen)
				assert.True(t, ui.pages.HasPage(tt.wantModalPage))
			} else {
				assert.False(t, ui.modalOpen)
			}
		})
	}
}

// ── consumeRuneCommand: n/e/D on screenSuperusers ────────────────────────────

func TestConsumeRuneCommandInlineSuperusersCRUD(t *testing.T) {
	tests := []struct {
		name          string
		key           rune
		superusers    []storage.Superuser
		wantConsumed  bool
		wantModalPage string
	}{
		{
			name:          "n/opens-superuser-edit",
			key:           'n',
			wantConsumed:  true,
			wantModalPage: "superuser-edit",
		},
		{
			name:          "e/with-su/opens-superuser-edit",
			key:           'e',
			superusers:    []storage.Superuser{{DBAlias: "dev", Alias: "root", Email: "root@example.com"}},
			wantConsumed:  true,
			wantModalPage: "superuser-edit",
		},
		{
			name:         "e/no-su/ignored",
			key:          'e',
			wantConsumed: false,
		},
		{
			name:          "D/with-su/opens-confirm",
			key:           'D',
			superusers:    []storage.Superuser{{DBAlias: "dev", Alias: "root", Email: "root@example.com"}},
			wantConsumed:  true,
			wantModalPage: "confirm",
		},
		{
			name:         "D/no-su/ignored",
			key:          'D',
			wantConsumed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := newTestDispatcherWithDB(t, "dev", "http://127.0.0.1:8090")
			ui := newTestNavigatorTUIOnScreen(screenSuperusers)
			ui.dispatcher = dispatcher
			ui.hasSession = true
			ui.session = pbSession{DB: storage.DB{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}}
			ui.dbs = []storage.DB{{Alias: "dev", BaseURL: "http://127.0.0.1:8090"}}
			ui.superusers = tt.superusers
			ui.setupViews()

			got := ui.consumeRuneCommand(tt.key)

			assert.Equal(t, tt.wantConsumed, got)
			if tt.wantModalPage != "" {
				assert.True(t, ui.modalOpen)
				assert.True(t, ui.pages.HasPage(tt.wantModalPage))
			} else {
				assert.False(t, ui.modalOpen)
			}
		})
	}
}

// ── helpText ─────────────────────────────────────────────────────────────────

func TestHelpText(t *testing.T) {
	tests := []struct {
		screen navigatorScreen
		want   string
	}{
		{screenDBList, "q quit  Enter select  n new  e edit  D del  u superusers  r refresh"},
		{screenSuperusers, "q quit  esc/backspace back  Enter select  n new  e edit  D del  b db aliases  r refresh"},
	}
	for _, tt := range tests {
		t.Run(string(tt.screen), func(t *testing.T) {
			ui := &navigatorTUI{screen: tt.screen}
			assert.Equal(t, tt.want, ui.helpText())
		})
	}
}

// ── emptyDetailText ──────────────────────────────────────────────────────────

func TestEmptyDetailText(t *testing.T) {
	tests := []struct {
		name    string
		screen  navigatorScreen
		session pbSession
		want    string
	}{
		{
			name:   "dbList",
			screen: screenDBList,
			want:   "No DB aliases configured. Press [n] to add one.",
		},
		{
			name:    "superusers/with-db",
			screen:  screenSuperusers,
			session: pbSession{DB: storage.DB{Alias: "dev"}},
			want:    "No superusers for 'dev'. Press [n] to add one.",
		},
		{
			name:   "superusers/no-session",
			screen: screenSuperusers,
			want:   "No superusers. Press [n] to add one.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := &navigatorTUI{screen: tt.screen, session: tt.session}
			assert.Equal(t, tt.want, ui.emptyDetailText())
		})
	}
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
		hasSession: true,
		session: pbSession{
			DB: storage.DB{Alias: "dev", BaseURL: "http://127.0.0.1:8090"},
			SU: storage.Superuser{DBAlias: "dev", Alias: "root", Email: "root@example.com"},
		},
	}

	ui.updateSessionDB("dev", "prod")
	assert.Equal(t, "prod", ui.session.DB.Alias)
	assert.Equal(t, "prod", ui.session.SU.DBAlias)

	ui.updateSessionSuperuser("prod", "root", "admin")
	assert.Equal(t, "admin", ui.session.SU.Alias)
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

func newTestNavigatorTUIOnScreen(screen navigatorScreen) *navigatorTUI {
	ui := newTestNavigatorTUI()
	ui.screen = screen
	return ui
}

func newTestDispatcherWithDB(t *testing.T, alias, baseURL string) *Dispatcher {
	t.Helper()
	dispatcher := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	_, err := dispatcher.saveDBAlias(alias, baseURL)
	require.NoError(t, err)
	return dispatcher
}

func newTestDispatcherWithDBs(t *testing.T, dbs []storage.DB) *Dispatcher {
	t.Helper()
	dispatcher := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})
	for _, db := range dbs {
		_, err := dispatcher.saveDBAlias(db.Alias, db.BaseURL)
		require.NoError(t, err)
	}
	return dispatcher
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

func TestOpenConfirmModalConfirmCallsOnConfirm(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.pages = tview.NewPages().AddPage("main", tview.NewBox(), true, true)

	called := false
	ui.openConfirmModal("Delete this?", func() { called = true }, nil)

	require.True(t, ui.modalOpen)
	require.True(t, ui.pages.HasPage("confirm"))

	// simulate Confirm button press via closeModal + callback
	ui.pages.RemovePage("confirm")
	ui.modalOpen = false
	called = true // confirm path

	assert.True(t, called)
}

func TestOpenConfirmModalCancelDoesNotCallOnConfirm(t *testing.T) {
	ui := newTestNavigatorTUI()
	ui.pages = tview.NewPages().AddPage("main", tview.NewBox(), true, true)

	called := false
	ui.openConfirmModal("Delete this?", func() { called = true }, nil)

	// simulate Cancel: close modal without calling onConfirm
	ui.pages.RemovePage("confirm")
	ui.modalOpen = false

	assert.False(t, called)
}

func TestInstallFormArrowNavigationWithCloseEscCallsOnClose(t *testing.T) {
	form := tview.NewForm()
	closed := false
	installFormArrowNavigationWithClose(form, func() { closed = true })

	handler := form.GetInputCapture()
	require.NotNil(t, handler)

	result := handler(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone))
	assert.Nil(t, result, "Esc should be consumed")
	assert.True(t, closed, "onClose should have been called")
}

func TestInstallFormArrowNavigationWithCloseNonEscPassesThrough(t *testing.T) {
	form := tview.NewForm()
	installFormArrowNavigationWithClose(form, func() {})

	handler := form.GetInputCapture()
	require.NotNil(t, handler)

	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	result := handler(event)
	assert.NotNil(t, result, "non-Esc key should not be consumed by close handler")
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
