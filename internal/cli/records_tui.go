package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/jiseop121/pbdash/internal/apperr"
	"github.com/jiseop121/pbdash/internal/pocketbase"
	"github.com/jiseop121/pbdash/internal/storage"
)

const (
	visibleColumnWindow = 8
	managerNewOption    = "<new>"
)

type navigatorScreen string

const (
	screenDBList      navigatorScreen = "dbs"
	screenSuperusers  navigatorScreen = "superusers"
	screenCollections navigatorScreen = "collections"
	screenRecords     navigatorScreen = "records"
	screenRecordDetail navigatorScreen = "record-detail"
)

type navigatorRoute struct {
	screen    navigatorScreen
	session    pbSession
	hasSession bool
	state     RecordsQueryState
}

type navigatorTUI struct {
	dispatcher *Dispatcher
	ctx        context.Context

	app    *tview.Application
	stop   func()
	pages  *tview.Pages
	layout *tview.Flex
	termScreen tcell.Screen

	statusView *tview.TextView
	tableView  *tview.Table
	detailView *tview.TextView
	helpView   *tview.TextView

	screen      navigatorScreen
	history     []navigatorScreen
	hasSession   bool
	session      pbSession
	dbs         []storage.DB
	superusers  []storage.Superuser
	collections []map[string]any

	recordsState  RecordsQueryState
	result        pocketbase.QueryResult
	recordDetail  map[string]any
	totalItems    int
	totalPages    int
	selectedIndex int
	columnOffset  int
	detailVisible bool
	observedCols  map[string]struct{}
	statusMessage string
}

type dbManagerState struct {
	items         []storage.DB
	selectedAlias string
}

type superuserManagerState struct {
	dbs           []storage.DB
	selectedDB    string
	superusers    []storage.Superuser
	selectedAlias string
}

func newDBManagerState(items []storage.DB) dbManagerState {
	return dbManagerState{items: items}
}

func (m *dbManagerState) selectAlias(value string) (storage.DB, bool) {
	m.selectedAlias = normalizeManagerSelection(value)
	return findDB(m.items, m.selectedAlias)
}

func (m dbManagerState) save(dispatcher *Dispatcher, alias, baseURL string) error {
	if m.selectedAlias == "" {
		_, err := dispatcher.saveDBAlias(alias, baseURL)
		return err
	}

	_, err := dispatcher.updateDBAlias(m.selectedAlias, alias, baseURL)
	return err
}

func (m dbManagerState) remove(dispatcher *Dispatcher) error {
	if m.selectedAlias == "" {
		return apperr.Invalid("Select an existing db alias first.", "Choose a saved db alias from the list.")
	}

	return dispatcher.removeDBAlias(m.selectedAlias)
}

func (m *superuserManagerState) selectAlias(value string) (storage.Superuser, bool) {
	m.selectedAlias = normalizeManagerSelection(value)
	return findSuperuser(m.superusers, m.selectedAlias)
}

func (m superuserManagerState) save(dispatcher *Dispatcher, alias, email, password string) error {
	if m.selectedAlias == "" {
		_, err := dispatcher.saveSuperuser(m.selectedDB, alias, email, password)
		return err
	}

	_, err := dispatcher.updateSuperuser(m.selectedDB, m.selectedAlias, alias, email, password)
	return err
}

func (m superuserManagerState) remove(dispatcher *Dispatcher) error {
	if m.selectedAlias == "" {
		return apperr.Invalid("Select an existing superuser first.", "Choose a saved superuser from the list.")
	}

	return dispatcher.removeSuperuser(m.selectedDB, m.selectedAlias)
}

func (d *Dispatcher) RunRecordsTUI(ctx context.Context, session pbSession, state RecordsQueryState) error {
	return d.runRecordsTUI(ctx, session, state)
}

func (d *Dispatcher) runRecordsTUI(ctx context.Context, session pbSession, state RecordsQueryState) error {
	return d.navigatorRunner(ctx, navigatorRoute{
		screen:    screenRecords,
		session:   session,
		hasSession: true,
		state:     state,
	})
}

func (d *Dispatcher) runNavigatorTUI(ctx context.Context, route navigatorRoute) (runErr error) {
	defer func() {
		if r := recover(); r != nil {
			runErr = apperr.RuntimeErr("TUI mode crashed unexpectedly.", "Please report this issue.", fmt.Errorf("%v", r))
		}
	}()

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	ui := &navigatorTUI{
		dispatcher:    d,
		ctx:           ctx,
		app:           tview.NewApplication(),
		statusView:    tview.NewTextView(),
		tableView:     tview.NewTable(),
		detailView:    tview.NewTextView(),
		helpView:      tview.NewTextView(),
		detailVisible: true,
		observedCols:  map[string]struct{}{},
	}
	ui.stop = ui.app.Stop
	if err := ui.bootstrap(route); err != nil {
		return err
	}
	ui.setupViews()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			ui.app.Stop()
		case <-done:
		}
	}()

	ui.app.SetRoot(ui.pages, true)
	ui.focusMain()

	err := ui.app.Run()
	close(done)
	if err != nil {
		return apperr.RuntimeErr("Could not run TUI mode.", "", err)
	}
	if ctx.Err() != nil {
		return apperr.RuntimeErr("TUI mode was interrupted.", "", ctx.Err())
	}
	return nil
}

func (ui *navigatorTUI) bootstrap(route navigatorRoute) error {
	if route.hasSession && route.screen == screenRecords {
		ui.hasSession = true
		ui.session = route.session
		ui.recordsState = route.state
		if ui.recordsState.Page == 0 {
			ui.recordsState.Page = 1
		}
		if err := ui.loadCollections(); err != nil {
			return err
		}
		ui.history = []navigatorScreen{screenCollections}
		ui.screen = screenRecords
		return ui.fetchRecords()
	}

	ui.screen = screenDBList
	return ui.loadDBs()
}

func (ui *navigatorTUI) setupViews() {
	ui.statusView.SetDynamicColors(true)
	ui.app.SetInputCapture(ui.handleGlobalKey)
	ui.app.SetAfterDrawFunc(func(screen tcell.Screen) {
		ui.termScreen = screen
	})

	ui.tableView.SetBorders(false)
	ui.tableView.SetSelectable(true, false)
	ui.tableView.SetFixed(1, 0)
	ui.tableView.SetInputCapture(ui.handleKey)
	ui.tableView.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	ui.tableView.SetSelectionChangedFunc(func(row, _ int) {
		if row <= 0 {
			ui.selectedIndex = 0
		} else {
			ui.selectedIndex = row - 1
		}
		if ui.shouldShowDetailPane() {
			ui.renderDetail()
		}
	})

	ui.detailView.SetTextAlign(tview.AlignLeft)
	ui.detailView.SetDynamicColors(true)
	ui.detailView.SetBorder(true)
	ui.detailView.SetScrollable(true)
	ui.detailView.SetInputCapture(ui.handleKey)

	ui.layout = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.layout.AddItem(ui.statusView, 1, 0, false)
	ui.layout.AddItem(ui.tableView, 0, 1, true)
	ui.layout.AddItem(ui.detailView, 9, 0, false)
	ui.layout.AddItem(ui.helpView, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", ui.layout, true, true)

	ui.renderCurrentScreen()
}

func (ui *navigatorTUI) handleGlobalKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.isModalOpen() || event == nil {
		return event
	}

	if ui.consumeGlobalKey(event) {
		return nil
	}
	return event
}

func (ui *navigatorTUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.isModalOpen() || event == nil {
		return event
	}

	if ui.consumeNavigationKey(event.Key()) {
		return nil
	}
	if ui.consumeRuneCommand(event.Rune()) {
		return nil
	}

	return event
}

func (ui *navigatorTUI) openFilterModal() {
	ui.openSubmitCancelInputModal("Filter", "filter", ui.recordsState.Filter, "name = 'foo' && active = true", func(val string) error {
		ui.recordsState.Filter = strings.TrimSpace(val)
		ui.recordsState.Page = 1
		return ui.fetchAndRenderRecords()
	})
}

func (ui *navigatorTUI) openSortModal() {
	ui.openSubmitCancelInputModal("Sort", "sort", ui.recordsState.Sort, "-created,+name", func(val string) error {
		ui.recordsState.Sort = strings.TrimSpace(val)
		ui.recordsState.Page = 1
		return ui.fetchAndRenderRecords()
	})
}

func (ui *navigatorTUI) consumeGlobalKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEsc, tcell.KeyBackspace, tcell.KeyBackspace2:
		if ui.screen == screenDBList {
			return true // 첫 화면에서 실수로 종료되는 것 방지
		}
		ui.goBack()
		return true
	}

	if event.Rune() == 'q' {
		ui.stopApplication()
		return true
	}
	return false
}

func (ui *navigatorTUI) consumeNavigationKey(key tcell.Key) bool {
	switch key {
	case tcell.KeyEnter:
		ui.handleEnter()
		return true
	case tcell.KeyLeft:
		return ui.shiftRecordsColumns(-1)
	case tcell.KeyRight:
		return ui.shiftRecordsColumns(1)
	default:
		return false
	}
}

func (ui *navigatorTUI) consumeRuneCommand(key rune) bool {
	switch key {
	case 'h':
		return ui.shiftRecordsColumns(-1)
	case 'l':
		return ui.shiftRecordsColumns(1)
	case '/':
		return ui.openRecordsAction(ui.openFilterModal)
	case 's':
		return ui.openRecordsAction(ui.openSortModal)
	case 'c':
		return ui.openRecordsAction(ui.openColumnsModal)
	case 'y':
		return ui.copyRecordDetail()
	case 'd':
		if ui.screen == screenRecords || ui.screen == screenRecordDetail ||
			ui.screen == screenDBList || ui.screen == screenSuperusers {
			return false
		}
		ui.detailVisible = !ui.detailVisible
		ui.renderCurrentScreen()
		return true
	case 'b':
		if ui.screen == screenDBList {
			return false
		}
		ui.openDBListModal()
		return true
	case 'u':
		if ui.screen == screenSuperusers {
			return false
		}
		if ui.screen == screenDBList {
			if len(ui.dbs) == 0 {
				return false
			}
			db := ui.dbs[ui.selectedIndex]
			ui.openSuperuserListModalForDB(db.Alias)
			return true
		}
		ui.openSuperuserListModal()
		return true
	case 'n':
		if ui.screen == screenDBList {
			ui.openDBEditModal(nil, func() { ui.closeModal("db-edit") })
			return true
		}
		if ui.screen == screenSuperusers {
			ui.openSuperuserEditModal(ui.session.DB.Alias, nil,
				func() { ui.closeModal("superuser-edit") })
			return true
		}
		return false
	case 'e':
		if ui.screen == screenDBList && len(ui.dbs) > 0 {
			db := ui.dbs[ui.selectedIndex]
			ui.openDBEditModal(&db, func() { ui.closeModal("db-edit") })
			return true
		}
		if ui.screen == screenSuperusers && len(ui.superusers) > 0 {
			su := ui.superusers[ui.selectedIndex]
			ui.openSuperuserEditModal(ui.session.DB.Alias, &su,
				func() { ui.closeModal("superuser-edit") })
			return true
		}
		return false
	case 'D':
		if ui.screen == screenDBList && len(ui.dbs) > 0 {
			manager := newDBManagerState(ui.dbs)
			manager.selectAlias(ui.dbs[ui.selectedIndex].Alias)
			ui.deleteDBManager(manager, nil)
			return true
		}
		if ui.screen == screenSuperusers && len(ui.superusers) > 0 {
			manager := newSuperuserManagerState(ui.dbs, ui.session.DB.Alias)
			manager.superusers = ui.superusers
			manager.selectAlias(ui.superusers[ui.selectedIndex].Alias)
			ui.deleteSuperuserManager(manager, nil)
			return true
		}
		return false
	case 'r':
		if ui.isRecordDetailScreen() {
			return false
		}
		if err := ui.refreshCurrentScreen(); err != nil {
			ui.showError(err)
		}
		return true
	case '[':
		return ui.moveToPreviousRecordsPage()
	case ']':
		return ui.moveToNextRecordsPage()
	case 'g':
		return ui.jumpToRecordsPage(1)
	case 'G':
		if ui.totalPages == 0 {
			return false
		}
		return ui.jumpToRecordsPage(ui.totalPages)
	default:
		return false
	}
}

func (ui *navigatorTUI) openRecordsAction(action func()) bool {
	if !ui.isRecordsScreen() {
		return false
	}
	action()
	return true
}

func (ui *navigatorTUI) shiftRecordsColumns(delta int) bool {
	if !ui.isRecordsScreen() {
		return false
	}
	ui.shiftColumns(delta)
	return true
}

func (ui *navigatorTUI) moveToPreviousRecordsPage() bool {
	if !ui.isRecordsScreen() || ui.recordsState.Page <= 1 {
		return false
	}
	ui.recordsState.Page--
	if err := ui.fetchAndRenderRecords(); err != nil {
		ui.showError(err)
	}
	return true
}

func (ui *navigatorTUI) moveToNextRecordsPage() bool {
	if !ui.isRecordsScreen() || (ui.totalPages > 0 && ui.recordsState.Page >= ui.totalPages) {
		return false
	}
	ui.recordsState.Page++
	if err := ui.fetchAndRenderRecords(); err != nil {
		ui.showError(err)
	}
	return true
}

func (ui *navigatorTUI) jumpToRecordsPage(page int) bool {
	if !ui.isRecordsScreen() || page <= 0 {
		return false
	}
	ui.recordsState.Page = page
	if err := ui.fetchAndRenderRecords(); err != nil {
		ui.showError(err)
	}
	return true
}

func (ui *navigatorTUI) isRecordsScreen() bool {
	return ui.screen == screenRecords
}

func (ui *navigatorTUI) isRecordDetailScreen() bool {
	return ui.screen == screenRecordDetail
}

func (ui *navigatorTUI) isModalOpen() bool {
	if ui.pages == nil {
		return false
	}
	return ui.pages.GetPageCount() > 1
}

func (ui *navigatorTUI) handleEnter() {
	switch ui.screen {
	case screenDBList:
		ui.setLoadingStatus()
		if err := ui.activateSelectedDB(); err != nil {
			ui.clearLoadingStatus()
			ui.showError(err)
		}
	case screenSuperusers:
		ui.setLoadingStatus()
		if err := ui.activateSelectedSuperuser(); err != nil {
			ui.clearLoadingStatus()
			ui.showError(err)
		}
	case screenCollections:
		ui.setLoadingStatus()
		if err := ui.activateSelectedCollection(); err != nil {
			ui.clearLoadingStatus()
			ui.showError(err)
		}
	case screenRecords:
		ui.setLoadingStatus()
		if err := ui.activateSelectedRecordDetail(); err != nil {
			ui.clearLoadingStatus()
			ui.showError(err)
		}
	}
}

func (ui *navigatorTUI) copyRecordDetail() bool {
	if !ui.isRecordDetailScreen() {
		return false
	}
	body, err := ui.recordDetailText()
	if err != nil {
		ui.showError(err)
		return true
	}
	if ui.termScreen == nil {
		ui.showError(apperr.RuntimeErr("Clipboard is not available.", "Try again after the TUI finishes drawing.", nil))
		return true
	}
	ui.termScreen.SetClipboard([]byte(body))
	ui.statusMessage = "copied (OSC52)"
	if ui.statusView != nil {
		ui.statusView.SetText(ui.statusText())
	}
	return true
}

func (ui *navigatorTUI) goBack() {
	if len(ui.history) == 0 {
		ui.app.Stop()
		return
	}
	last := ui.history[len(ui.history)-1]
	ui.history = ui.history[:len(ui.history)-1]
	ui.screen = last
	ui.selectedIndex = 0
	ui.columnOffset = 0
	ui.renderCurrentScreen()
}

func (ui *navigatorTUI) refreshCurrentScreen() error {
	switch ui.screen {
	case screenDBList:
		ui.setLoadingStatus()
		if err := ui.loadDBs(); err != nil {
			ui.clearLoadingStatus()
			ui.showError(err)
			return err
		}
		ui.clearLoadingStatus()
	case screenSuperusers:
		ui.setLoadingStatus()
		if err := ui.loadSuperusers(); err != nil {
			ui.clearLoadingStatus()
			ui.showError(err)
			return err
		}
		ui.clearLoadingStatus()
	case screenCollections:
		ui.setLoadingStatus()
		if err := ui.loadCollections(); err != nil {
			ui.clearLoadingStatus()
			ui.showError(err)
			return err
		}
		ui.clearLoadingStatus()
	case screenRecords:
		return ui.fetchAndRenderRecords()
	}
	ui.renderCurrentScreen()
	return nil
}

func (ui *navigatorTUI) activateSelectedDB() error {
	if len(ui.dbs) == 0 || ui.selectedIndex >= len(ui.dbs) {
		return nil
	}

	db := ui.dbs[ui.selectedIndex]
	superusers, err := ui.dispatcher.suStore.ListByDB(db.Alias)
	if err != nil {
		return mapStoreError(err)
	}
	previousSession := ui.dispatcher.sessionCtx

	ui.hasSession = true
	ui.session = pbSession{DB: db}
	ui.dispatcher.sessionCtx.DBAlias = db.Alias
	ui.dispatcher.sessionCtx.SuperuserAlias = ""

	if preferred, ok := pickPreferredSuperuser(db.Alias, superusers, previousSession, ui.dispatcher.savedCtx); ok {
		ui.session.SU = preferred
		ui.dispatcher.sessionCtx.SuperuserAlias = preferred.Alias
		if err := ui.loadCollections(); err != nil {
			return err
		}
		ui.pushScreen(screenCollections)
		return nil
	}

	if len(superusers) == 0 {
		return apperr.Invalid("No superuser is configured for db \""+db.Alias+"\".", "Run `pbdash -c \"superuser add --db "+db.Alias+" ...\"` first.")
	}

	ui.superusers = superusers
	ui.pushScreen(screenSuperusers)
	return nil
}

func (ui *navigatorTUI) activateSelectedSuperuser() error {
	if len(ui.superusers) == 0 || ui.selectedIndex >= len(ui.superusers) {
		return nil
	}
	su := ui.superusers[ui.selectedIndex]
	ui.session.SU = su
	ui.dispatcher.sessionCtx = commandContext{DBAlias: ui.session.DB.Alias, SuperuserAlias: su.Alias}
	if err := ui.loadCollections(); err != nil {
		return err
	}
	ui.pushScreen(screenCollections)
	return nil
}

func (ui *navigatorTUI) activateSelectedCollection() error {
	rows := ui.collectionRows()
	if len(rows) == 0 || ui.selectedIndex >= len(rows) {
		return nil
	}
	name := collectionName(rows[ui.selectedIndex])
	if name == "" {
		return apperr.RuntimeErr("Could not determine collection name.", "", nil)
	}
	ui.recordsState = RecordsQueryState{Collection: name, Page: 1}
	ui.recordDetail = nil
	ui.observedCols = map[string]struct{}{}
	if err := ui.fetchRecords(); err != nil {
		return err
	}
	ui.pushScreen(screenRecords)
	return nil
}

func (ui *navigatorTUI) activateSelectedRecordDetail() error {
	row, ok := ui.selectedRecordRow()
	if !ok {
		return nil
	}
	ui.recordDetail = cloneRow(row)
	ui.pushScreen(screenRecordDetail)
	return nil
}

func (ui *navigatorTUI) pushScreen(next navigatorScreen) {
	ui.history = append(ui.history, ui.screen)
	ui.screen = next
	ui.selectedIndex = 0
	ui.columnOffset = 0
	ui.statusMessage = ""
	ui.renderCurrentScreen()
}

func (ui *navigatorTUI) loadDBs() error {
	items, err := ui.dispatcher.dbStore.List()
	if err != nil {
		return mapStoreError(err)
	}
	ui.dbs = items
	return nil
}

func (ui *navigatorTUI) loadSuperusers() error {
	if !ui.hasSession {
		return apperr.RuntimeErr("No db is selected.", "", nil)
	}
	items, err := ui.dispatcher.suStore.ListByDB(ui.session.DB.Alias)
	if err != nil {
		return mapStoreError(err)
	}
	ui.superusers = items
	return nil
}

func (ui *navigatorTUI) loadCollections() error {
	if !ui.hasSession {
		return apperr.RuntimeErr("No database selected.", "", nil)
	}
	result, err := ui.dispatcher.fetchCollections(ui.ctx, ui.session)
	if err != nil {
		return err
	}
	ui.collections = result.Rows
	return nil
}

func (ui *navigatorTUI) fetchRecords() error {
	result, err := ui.dispatcher.fetchRecords(ui.ctx, ui.session, ui.recordsState)
	if err != nil {
		return err
	}
	ui.result = result
	if result.Meta != nil {
		if result.Meta.Page > 0 {
			ui.recordsState.Page = result.Meta.Page
		}
		if result.Meta.PerPage > 0 {
			ui.recordsState.PerPage = result.Meta.PerPage
		}
		ui.totalPages = result.Meta.TotalPages
		ui.totalItems = result.Meta.TotalItems
	} else {
		ui.totalPages = 0
		ui.totalItems = len(result.Rows)
	}
	if ui.recordsState.Page == 0 {
		ui.recordsState.Page = 1
	}
	ui.observeColumns()
	return nil
}

func (ui *navigatorTUI) fetchAndRenderRecords() error {
	ui.setLoadingStatus()
	if err := ui.fetchRecords(); err != nil {
		ui.clearLoadingStatus()
		ui.showError(err)
		return err
	}
	ui.clearLoadingStatus()
	ui.renderCurrentScreen()
	return nil
}

func (ui *navigatorTUI) observeColumns() {
	fresh := pocketbase.CollectColumns(ui.result.Rows)
	mergeColumns(ui.observedCols, fresh)
}

func (ui *navigatorTUI) stopApplication() {
	if ui.stop != nil {
		ui.stop()
		return
	}
	if ui.app != nil {
		ui.app.Stop()
	}
}

func (ui *navigatorTUI) syncLocalConfigState() error {
	if err := ui.loadDBs(); err != nil {
		return err
	}
	if !ui.hasSession {
		return nil
	}

	db, found, err := ui.dispatcher.dbStore.Find(ui.session.DB.Alias)
	if err != nil {
		return mapStoreError(err)
	}
	if !found {
		ui.resetToDBList()
		return nil
	}
	ui.session.DB = db

	if strings.TrimSpace(ui.session.SU.Alias) != "" {
		su, found, err := ui.dispatcher.suStore.Find(db.Alias, ui.session.SU.Alias)
		if err != nil {
			return mapStoreError(err)
		}
		if found {
			ui.session.SU = su
		} else {
			ui.session.SU = storage.Superuser{}
			if ui.screen == screenCollections || ui.screen == screenRecords {
				ui.screen = screenSuperusers
				ui.history = []navigatorScreen{screenDBList}
			}
		}
	}

	switch ui.screen {
	case screenSuperusers:
		return ui.loadSuperusers()
	case screenCollections:
		if strings.TrimSpace(ui.session.SU.Alias) == "" {
			ui.screen = screenSuperusers
			return ui.loadSuperusers()
		}
		return ui.loadCollections()
	case screenRecords:
		if strings.TrimSpace(ui.session.SU.Alias) == "" {
			ui.screen = screenSuperusers
			return ui.loadSuperusers()
		}
		if err := ui.loadCollections(); err != nil {
			return err
		}
		return ui.fetchRecords()
	case screenRecordDetail:
		if strings.TrimSpace(ui.session.SU.Alias) == "" {
			ui.screen = screenSuperusers
			return ui.loadSuperusers()
		}
		if err := ui.loadCollections(); err != nil {
			return err
		}
		if err := ui.fetchRecords(); err != nil {
			return err
		}
		if row, ok := ui.selectedRecordRow(); ok {
			ui.recordDetail = cloneRow(row)
		}
		return nil
	default:
		return nil
	}
}

func (ui *navigatorTUI) resetToDBList() {
	ui.screen = screenDBList
	ui.history = nil
	ui.hasSession = false
	ui.session = pbSession{}
	ui.superusers = nil
	ui.collections = nil
	ui.result = pocketbase.QueryResult{}
	ui.recordDetail = nil
	ui.totalItems = 0
	ui.totalPages = 0
	ui.selectedIndex = 0
	ui.columnOffset = 0
}

func (ui *navigatorTUI) updateSessionDB(previousAlias, nextAlias string) {
	if !ui.hasSession || strings.TrimSpace(previousAlias) == "" {
		return
	}
	if !strings.EqualFold(ui.session.DB.Alias, previousAlias) {
		return
	}

	ui.session.DB.Alias = nextAlias
	if strings.EqualFold(ui.session.SU.DBAlias, previousAlias) {
		ui.session.SU.DBAlias = nextAlias
	}
}

func (ui *navigatorTUI) updateSessionSuperuser(dbAlias, previousAlias, nextAlias string) {
	if !ui.hasSession || strings.TrimSpace(previousAlias) == "" {
		return
	}
	if !strings.EqualFold(ui.session.DB.Alias, dbAlias) {
		return
	}
	if !strings.EqualFold(ui.session.SU.Alias, previousAlias) {
		return
	}

	ui.session.SU.Alias = nextAlias
}

func pickPreferredSuperuser(dbAlias string, candidates []storage.Superuser, sessionCtx, savedCtx commandContext) (storage.Superuser, bool) {
	if preferred, ok := matchContextSuperuser(dbAlias, candidates, sessionCtx); ok {
		return preferred, true
	}
	if preferred, ok := matchContextSuperuser(dbAlias, candidates, savedCtx); ok {
		return preferred, true
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}
	return storage.Superuser{}, false
}

func matchContextSuperuser(dbAlias string, candidates []storage.Superuser, ctx commandContext) (storage.Superuser, bool) {
	if !strings.EqualFold(strings.TrimSpace(ctx.DBAlias), strings.TrimSpace(dbAlias)) {
		return storage.Superuser{}, false
	}
	alias := strings.TrimSpace(ctx.SuperuserAlias)
	if alias == "" {
		return storage.Superuser{}, false
	}
	for _, item := range candidates {
		if strings.EqualFold(item.Alias, alias) {
			return item, true
		}
	}
	return storage.Superuser{}, false
}

func cloneRow(row map[string]any) map[string]any {
	if row == nil {
		return nil
	}
	out := make(map[string]any, len(row))
	for key, value := range row {
		out[key] = value
	}
	return out
}
