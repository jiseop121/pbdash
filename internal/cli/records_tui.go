package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	modalOpen   bool
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

func (d *Dispatcher) runNavigatorTUI(ctx context.Context, route navigatorRoute) error {
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
	if ui.modalOpen || event == nil {
		return event
	}

	if ui.consumeGlobalKey(event) {
		return nil
	}
	return event
}

func (ui *navigatorTUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.modalOpen || event == nil {
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
		_ = ui.refreshCurrentScreen()
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
	_ = ui.fetchAndRenderRecords()
	return true
}

func (ui *navigatorTUI) moveToNextRecordsPage() bool {
	if !ui.isRecordsScreen() || (ui.totalPages > 0 && ui.recordsState.Page >= ui.totalPages) {
		return false
	}
	ui.recordsState.Page++
	_ = ui.fetchAndRenderRecords()
	return true
}

func (ui *navigatorTUI) jumpToRecordsPage(page int) bool {
	if !ui.isRecordsScreen() || page <= 0 {
		return false
	}
	ui.recordsState.Page = page
	_ = ui.fetchAndRenderRecords()
	return true
}

func (ui *navigatorTUI) isRecordsScreen() bool {
	return ui.screen == screenRecords
}

func (ui *navigatorTUI) isRecordDetailScreen() bool {
	return ui.screen == screenRecordDetail
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

func (d *Dispatcher) fetchCollections(ctx context.Context, session pbSession) (pocketbase.QueryResult, error) {
	payload, err := d.getJSONWithAuth(ctx, session, pocketbase.BuildCollectionsEndpoint(), nil)
	if err != nil {
		return pocketbase.QueryResult{}, err
	}
	return pocketbase.ParseItemsResult(payload), nil
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

func (ui *navigatorTUI) currentColumns() []string {
	switch ui.screen {
	case screenDBList:
		return []string{"db_alias", "base_url"}
	case screenSuperusers:
		return []string{"superuser_alias", "email"}
	case screenCollections:
		return []string{"name"}
	case screenRecords:
		if len(ui.recordsState.Fields) > 0 {
			return ui.recordsState.Fields
		}
		cols := pocketbase.CollectColumns(ui.result.Rows)
		if len(cols) == 0 {
			return []string{"result"}
		}
		return cols
	case screenRecordDetail:
		return nil
	default:
		return []string{"result"}
	}
}

func (ui *navigatorTUI) currentRows() []map[string]any {
	switch ui.screen {
	case screenDBList:
		return ui.dbRows()
	case screenSuperusers:
		return ui.superuserRows()
	case screenCollections:
		return ui.collectionRows()
	case screenRecords:
		return ui.result.Rows
	case screenRecordDetail:
		return nil
	default:
		return nil
	}
}

func (ui *navigatorTUI) dbRows() []map[string]any {
	rows := make([]map[string]any, 0, len(ui.dbs))
	for _, item := range ui.dbs {
		rows = append(rows, map[string]any{
			"db_alias": item.Alias,
			"base_url": item.BaseURL,
		})
	}
	return rows
}

func (ui *navigatorTUI) superuserRows() []map[string]any {
	rows := make([]map[string]any, 0, len(ui.superusers))
	for _, item := range ui.superusers {
		rows = append(rows, map[string]any{
			"superuser_alias": item.Alias,
			"email":           item.Email,
		})
	}
	return rows
}

func (ui *navigatorTUI) collectionRows() []map[string]any {
	rows := make([]map[string]any, 0, len(ui.collections))
	for _, item := range ui.collections {
		row := make(map[string]any, len(item)+1)
		for key, value := range item {
			row[key] = value
		}
		row["name"] = collectionName(item)
		rows = append(rows, row)
	}
	return rows
}

func (ui *navigatorTUI) visibleColumns() []string {
	all := ui.currentColumns()
	if len(all) == 0 {
		return []string{"result"}
	}
	if ui.screen != screenRecords {
		return all
	}
	if ui.columnOffset >= len(all) {
		ui.columnOffset = len(all) - 1
		if ui.columnOffset < 0 {
			ui.columnOffset = 0
		}
	}
	start := ui.columnOffset
	end := len(all)
	if end-start > visibleColumnWindow {
		end = start + visibleColumnWindow
	}
	return all[start:end]
}

func (ui *navigatorTUI) renderCurrentScreen() {
	ui.statusView.SetText(ui.statusText())
	ui.helpView.SetText(ui.helpText())
	ui.detailView.SetTitle(" " + ui.detailTitle() + " ")
	if ui.isRecordDetailScreen() {
		ui.renderRecordDetailScreen()
		ui.focusMain()
		return
	}
	ui.renderNavigatorScreen()
	ui.focusMain()
}

func (ui *navigatorTUI) renderNavigatorScreen() {
	ui.layout.ResizeItem(ui.tableView, 0, 1)
	if ui.shouldShowDetailPane() {
		ui.layout.ResizeItem(ui.detailView, 9, 0)
		ui.renderDetail()
	} else {
		ui.layout.ResizeItem(ui.detailView, 0, 0)
		ui.detailView.SetText("")
	}
	ui.renderTable()
}

func (ui *navigatorTUI) renderRecordDetailScreen() {
	ui.layout.ResizeItem(ui.tableView, 0, 0)
	ui.layout.ResizeItem(ui.detailView, 0, 1)
	body, err := ui.recordDetailText()
	if err != nil {
		ui.detailView.SetText(formatValue(ui.recordDetail))
		return
	}
	ui.detailView.SetText(body)
}

func (ui *navigatorTUI) renderTable() {
	ui.tableView.Clear()
	rows := ui.currentRows()
	cols := ui.visibleColumns()
	for c, col := range cols {
		cell := tview.NewTableCell(strings.ToUpper(strings.ReplaceAll(col, "_", " ")))
		cell.SetSelectable(false)
		cell.SetAttributes(tcell.AttrBold)
		ui.tableView.SetCell(0, c, cell)
	}
	for r, row := range rows {
		for c, col := range cols {
			ui.tableView.SetCell(r+1, c, tview.NewTableCell(formatValue(row[col])))
		}
	}
	if len(rows) == 0 {
		ui.selectedIndex = 0
		msg := ui.emptyTableMessage()
		cell := tview.NewTableCell(msg).SetSelectable(false)
		ui.tableView.SetCell(1, 0, cell)
		return
	}
	if ui.selectedIndex >= len(rows) {
		ui.selectedIndex = len(rows) - 1
	}
	ui.tableView.Select(ui.selectedIndex+1, 0)
}

func (ui *navigatorTUI) renderDetail() {
	row, ok := ui.selectedRow()
	if !ok {
		ui.detailView.SetText(ui.emptyDetailText())
		return
	}
	body, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		ui.detailView.SetText(formatValue(row))
		return
	}
	ui.detailView.SetText(string(body))
}

func (ui *navigatorTUI) recordDetailText() (string, error) {
	if ui.recordDetail == nil {
		return ui.emptyDetailText(), nil
	}
	body, err := json.MarshalIndent(ui.recordDetail, "", "  ")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (ui *navigatorTUI) selectedRow() (map[string]any, bool) {
	rows := ui.currentRows()
	if len(rows) == 0 {
		return nil, false
	}
	if ui.selectedIndex < 0 || ui.selectedIndex >= len(rows) {
		ui.selectedIndex = 0
	}
	return rows[ui.selectedIndex], true
}

func (ui *navigatorTUI) selectedRecordRow() (map[string]any, bool) {
	if len(ui.result.Rows) == 0 {
		return nil, false
	}
	if ui.selectedIndex < 0 || ui.selectedIndex >= len(ui.result.Rows) {
		ui.selectedIndex = 0
	}
	return ui.result.Rows[ui.selectedIndex], true
}

func (ui *navigatorTUI) emptyTableMessage() string {
	if ui.screen == screenRecords && strings.TrimSpace(ui.recordsState.Filter) != "" {
		return "No records match filter: " + ui.recordsState.Filter
	}
	return ui.emptyDetailText()
}

func (ui *navigatorTUI) emptyDetailText() string {
	switch ui.screen {
	case screenDBList:
		return tview.Escape("No DB aliases configured. Press [n] to add one.")
	case screenSuperusers:
		db := strings.TrimSpace(ui.session.DB.Alias)
		if db == "" {
			return tview.Escape("No superusers. Press [n] to add one.")
		}
		return tview.Escape("No superusers for '" + db + "'. Press [n] to add one.")
	case screenCollections:
		return "No collections"
	case screenRecords:
		return "No records"
	case screenRecordDetail:
		return "No record detail"
	default:
		return "No data"
	}
}


func (ui *navigatorTUI) shiftColumns(delta int) {
	if ui.screen != screenRecords || delta == 0 {
		return
	}
	if delta < 0 {
		if ui.columnOffset == 0 {
			return
		}
		ui.columnOffset--
		ui.renderTable()
		return
	}
	all := ui.currentColumns()
	if ui.columnOffset+visibleColumnWindow >= len(all) {
		return
	}
	ui.columnOffset++
	ui.renderTable()
}

func (ui *navigatorTUI) statusText() string {
	const sep = "  │  "
	parts := []string{"path=" + ui.breadcrumb()}
	if ui.hasSession {
		parts = append(parts, "db="+ui.session.DB.Alias)
		if strings.TrimSpace(ui.session.SU.Alias) != "" {
			parts = append(parts, "superuser="+ui.session.SU.Alias)
		}
	}
	if ui.screen == screenRecords || ui.screen == screenRecordDetail {
		parts = append(parts, fmt.Sprintf("collection=%s", ui.recordsState.Collection))
		if ui.totalPages > 0 {
			parts = append(parts, fmt.Sprintf("page %d/%d (%d items)", ui.recordsState.Page, ui.totalPages, ui.totalItems))
		} else {
			parts = append(parts, fmt.Sprintf("page %d (%d items)", ui.recordsState.Page, ui.totalItems))
		}
		if strings.TrimSpace(ui.recordsState.Filter) != "" {
			parts = append(parts, fmt.Sprintf("filter=%q", ui.recordsState.Filter))
		}
		if strings.TrimSpace(ui.recordsState.Sort) != "" {
			parts = append(parts, fmt.Sprintf("sort=%q", ui.recordsState.Sort))
		}
	}
	if strings.TrimSpace(ui.statusMessage) != "" {
		parts = append(parts, ui.statusMessage)
	}
	return strings.Join(parts, sep)
}

func (ui *navigatorTUI) breadcrumb() string {
	trail := []string{"dbs"}
	if ui.hasSession {
		trail = append(trail, ui.session.DB.Alias)
	}
	if ui.screen == screenSuperusers || ui.screen == screenCollections || ui.screen == screenRecords || ui.screen == screenRecordDetail {
		if strings.TrimSpace(ui.session.SU.Alias) != "" {
			trail = append(trail, ui.session.SU.Alias)
		} else if ui.screen == screenSuperusers {
			trail = append(trail, "superusers")
		}
	}
	if ui.screen == screenCollections || ui.screen == screenRecords || ui.screen == screenRecordDetail {
		trail = append(trail, "collections")
	}
	if (ui.screen == screenRecords || ui.screen == screenRecordDetail) && strings.TrimSpace(ui.recordsState.Collection) != "" {
		trail = append(trail, ui.recordsState.Collection)
	}
	if ui.screen == screenRecordDetail {
		trail = append(trail, "record")
	}
	return strings.Join(trail, " > ")
}

func (ui *navigatorTUI) helpText() string {
	switch ui.screen {
	case screenDBList:
		return "q quit  Enter select  n new  e edit  D del  u superusers  r refresh"
	case screenSuperusers:
		return "q quit  esc/backspace back  Enter select  n new  e edit  D del  b db aliases  r refresh"
	case screenCollections:
		return "q quit  esc/backspace back  Enter select  d detail  b db aliases  u superusers  r refresh"
	case screenRecords:
		return "q quit  esc/backspace back  h/l or <-/-> horiz  / filter  s sort  c columns  b db aliases  u superusers  [/] page  g/G first/last  r refresh  Enter detail"
	case screenRecordDetail:
		return "q quit  esc/backspace back  y copy  b db aliases  u superusers"
	default:
		return "q quit"
	}
}

func (ui *navigatorTUI) detailTitle() string {
	switch ui.screen {
	case screenDBList:
		return "db detail"
	case screenSuperusers:
		return "superuser detail"
	case screenCollections:
		return "collection detail"
	case screenRecords:
		return "record detail"
	case screenRecordDetail:
		return "record detail"
	default:
		return "detail"
	}
}

func (ui *navigatorTUI) shouldShowDetailPane() bool {
	return ui.screen != screenDBList &&
		ui.screen != screenSuperusers &&
		ui.screen != screenRecords &&
		ui.screen != screenRecordDetail &&
		ui.detailVisible
}

func (ui *navigatorTUI) openInputModal(title, label, current string, apply func(string) error) {
	ui.modalOpen = true
	form := tview.NewForm()
	form.AddInputField(label, current, 0, nil, nil)
	form.AddButton("Apply", func() {
		value := form.GetFormItem(0).(*tview.InputField).GetText()
		ui.closeModal("input")
		if err := apply(value); err != nil {
			ui.showError(err)
		}
	})
	form.AddButton("Cancel", func() {
		ui.closeModal("input")
	})
	form.SetBorder(true).SetTitle(" " + title + " ")
	form.SetButtonsAlign(tview.AlignRight)
	installFormArrowNavigation(form)

	modal := center(60, 9, form)
	ui.pages.AddPage("input", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) openSubmitCancelInputModal(title, label, current, placeholder string, apply func(string) error) {
	ui.modalOpen = true
	form := tview.NewForm()
	form.AddInputField(label, current, 0, nil, nil)
	form.GetFormItem(0).(*tview.InputField).SetPlaceholder(placeholder)
	form.SetBorder(true).SetTitle(" " + title + " ")
	installSubmitCancelNavigation(form, func() {
		value := form.GetFormItem(0).(*tview.InputField).GetText()
		ui.closeModal("input")
		if err := apply(value); err != nil {
			ui.showError(err)
		}
	}, func() {
		ui.closeModal("input")
	})

	modal := center(60, 7, form)
	ui.pages.AddPage("input", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) openColumnsModal() {
	cols := mergeColumns(ui.observedCols, nil)
	if len(cols) == 0 {
		ui.showError(apperr.Invalid("No columns available yet.", "Refresh results first."))
		return
	}

	selected := map[string]bool{}
	if len(ui.recordsState.Fields) == 0 {
		for _, col := range cols {
			selected[col] = true
		}
	} else {
		for _, col := range ui.recordsState.Fields {
			selected[col] = true
		}
	}

	ui.modalOpen = true
	form := tview.NewForm()
	for _, col := range cols {
		name := col
		form.AddCheckbox(name, selected[name], func(checked bool) {
			selected[name] = checked
		})
	}
	installSubmitCancelNavigation(form, func() {
		picked := make([]string, 0, len(selected))
		for _, col := range cols {
			if selected[col] {
				picked = append(picked, col)
			}
		}
		if len(picked) == 0 {
			ui.showError(apperr.Invalid("At least one column must be selected.", "Choose one or more columns."))
			return
		}
		ui.recordsState.Fields = normalizeColumns(picked)
		ui.recordsState.Page = 1
		ui.closeModal("columns")
		_ = ui.fetchAndRenderRecords()
	}, func() {
		ui.closeModal("columns")
	})
	form.SetBorder(true).SetTitle(" Columns ")

	height := len(cols) + 3
	if height > 24 {
		height = 24
	}
	modal := center(70, height, form)
	ui.pages.AddPage("columns", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) openDBListModal() {
	const pageName = "db-list"
	items, err := ui.dispatcher.dbStore.List()
	if err != nil {
		ui.showError(mapStoreError(err))
		return
	}

	table := buildManagerTable(dbListRows(items))
	closeFn := func() { ui.closeModal(pageName) }
	var btnForm *tview.Form

	table.SetSelectedFunc(func(row, _ int) {
		if row >= 0 && row < len(items) {
			item := items[row]
			ui.pages.RemovePage(pageName)
			ui.openDBEditModal(&item, func() {
				ui.pages.RemovePage("db-edit")
				ui.openDBListModal()
			})
		}
	})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			closeFn()
			return nil
		case tcell.KeyTab:
			ui.app.SetFocus(btnForm)
			return nil
		}
		switch event.Rune() {
		case 'n':
			ui.pages.RemovePage(pageName)
			ui.openDBEditModal(nil, func() {
				ui.pages.RemovePage("db-edit")
				ui.openDBListModal()
			})
			return nil
		case 'e':
			if len(items) == 0 {
				return nil
			}
			row, _ := table.GetSelection()
			if row < 0 || row >= len(items) {
				return nil
			}
			item := items[row]
			ui.pages.RemovePage(pageName)
			ui.openDBEditModal(&item, func() {
				ui.pages.RemovePage("db-edit")
				ui.openDBListModal()
			})
			return nil
		case 'D':
			if len(items) == 0 {
				return nil
			}
			row, _ := table.GetSelection()
			if row < 0 || row >= len(items) {
				return nil
			}
			manager := newDBManagerState(items)
			manager.selectAlias(items[row].Alias)
			ui.deleteDBManager(manager, func() { ui.openDBListModal() })
			return nil
		}
		return event
	})

	btnForm = tview.NewForm()
	btnForm.AddButton("New", func() {
		ui.pages.RemovePage(pageName)
		ui.openDBEditModal(nil, func() {
			ui.pages.RemovePage("db-edit")
			ui.openDBListModal()
		})
	})
	btnForm.AddButton("Edit", func() {
		if len(items) == 0 {
			return
		}
		row, _ := table.GetSelection()
		if row < 0 || row >= len(items) {
			return
		}
		item := items[row]
		ui.pages.RemovePage(pageName)
		ui.openDBEditModal(&item, func() {
			ui.pages.RemovePage("db-edit")
			ui.openDBListModal()
		})
	})
	btnForm.AddButton("Delete", func() {
		if len(items) == 0 {
			return
		}
		row, _ := table.GetSelection()
		if row < 0 || row >= len(items) {
			return
		}
		manager := newDBManagerState(items)
		manager.selectAlias(items[row].Alias)
		ui.deleteDBManager(manager, func() { ui.openDBListModal() })
	})
	btnForm.AddButton("Close", closeFn)
	btnForm.SetButtonsAlign(tview.AlignRight)
	btnForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyBacktab:
			if len(items) > 0 {
				ui.app.SetFocus(table)
				return nil
			}
		case tcell.KeyEsc:
			closeFn()
			return nil
		}
		return remapFormArrowNavigation(currentFormPrimitive(btnForm), event)
	})

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true).SetTitle(" DB Aliases ")
	container.AddItem(table, 0, 1, len(items) > 0)
	container.AddItem(btnForm, 3, 0, len(items) == 0)

	ui.modalOpen = true
	ui.pages.AddPage(pageName, center(80, 16, container), true, true)
	if len(items) > 0 {
		ui.app.SetFocus(table)
	} else {
		ui.app.SetFocus(btnForm)
	}
}

func (ui *navigatorTUI) openDBEditModal(item *storage.DB, onBack func()) {
	const pageName = "db-edit"
	isNew := item == nil

	items, err := ui.dispatcher.dbStore.List()
	if err != nil {
		ui.showError(mapStoreError(err))
		return
	}
	manager := newDBManagerState(items)

	var title, aliasValue, baseURLValue, submitLabel string
	if isNew {
		title = " New DB Alias "
		submitLabel = "Add"
	} else {
		manager.selectAlias(item.Alias)
		title = " Edit: " + item.Alias + " "
		aliasValue = item.Alias
		baseURLValue = item.BaseURL
		submitLabel = "Update"
	}

	backFn := func() {
		ui.pages.RemovePage(pageName)
		onBack()
	}

	form := tview.NewForm()
	form.AddInputField("alias", aliasValue, 0, nil, nil)
	form.AddInputField("base url", baseURLValue, 0, nil, nil)

	aliasField := form.GetFormItem(0).(*tview.InputField)
	baseURLField := form.GetFormItem(1).(*tview.InputField)

	if isNew {
		aliasField.SetPlaceholder("my-app")
		baseURLField.SetPlaceholder("https://my-app.pockethost.io")
	}

	form.AddButton(submitLabel, func() {
		ui.saveDBManager(manager, aliasField.GetText(), baseURLField.GetText())
	})
	form.AddButton("Back", backFn)
	form.SetBorder(true).SetTitle(title)
	form.SetButtonsAlign(tview.AlignRight)
	installFormArrowNavigationWithClose(form, backFn)

	ui.modalOpen = true
	ui.pages.AddPage(pageName, center(72, 10, form), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) openSuperuserListModal() {
	ui.openSuperuserListModalForDB(ui.session.DB.Alias)
}

func (ui *navigatorTUI) openSuperuserListModalForDB(initialDB string) {
	const pageName = "superuser-list"
	dbs, err := ui.dispatcher.dbStore.List()
	if err != nil {
		ui.showError(mapStoreError(err))
		return
	}
	if len(dbs) == 0 {
		ui.showError(apperr.Invalid("No db aliases are configured.", "Save a db alias before adding superusers."))
		return
	}

	manager := newSuperuserManagerState(dbs, initialDB)
	if err := manager.loadSuperusers(ui.dispatcher); err != nil {
		ui.showError(err)
		return
	}

	table := tview.NewTable().SetSelectable(true, false)
	table.SetBorderPadding(0, 0, 1, 1)

	fillSuperuserTable := func() {
		table.Clear()
		if len(manager.superusers) == 0 {
			table.SetCell(0, 0, tview.NewTableCell("No entries yet. Press New to add one."))
			table.SetSelectable(false, false)
			return
		}
		table.SetSelectable(true, false)
		for i, su := range manager.superusers {
			table.SetCell(i, 0, tview.NewTableCell(su.Alias).SetAttributes(tcell.AttrBold).SetExpansion(1).SetMaxWidth(0))
			table.SetCell(i, 1, tview.NewTableCell(su.Email).SetExpansion(1).SetMaxWidth(0))
		}
	}
	fillSuperuserTable()

	closeFn := func() { ui.closeModal(pageName) }

	openEditForRow := func(row int) {
		if row < 0 || row >= len(manager.superusers) {
			return
		}
		su := manager.superusers[row]
		ui.pages.RemovePage(pageName)
		ui.openSuperuserEditModal(manager.selectedDB, &su, func() {
			ui.pages.RemovePage("superuser-edit")
			ui.openSuperuserListModal()
		})
	}

	table.SetSelectedFunc(func(row, _ int) {
		openEditForRow(row)
	})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			closeFn()
			return nil
		}
		switch event.Rune() {
		case 'n':
			ui.pages.RemovePage(pageName)
			ui.openSuperuserEditModal(manager.selectedDB, nil, func() {
				ui.pages.RemovePage("superuser-edit")
				ui.openSuperuserListModal()
			})
			return nil
		case 'e':
			row, _ := table.GetSelection()
			openEditForRow(row)
			return nil
		case 'D':
			if len(manager.superusers) == 0 {
				return nil
			}
			row, _ := table.GetSelection()
			if row < 0 || row >= len(manager.superusers) {
				return nil
			}
			manager.selectAlias(manager.superusers[row].Alias)
			ui.deleteSuperuserManager(manager, func() { ui.openSuperuserListModalForDB(manager.selectedDB) })
			return nil
		}
		return event
	})

	dbForm := tview.NewForm()
	dbForm.AddDropDown("db", dbAliasOptions(dbs), manager.selectedDBIndex(), func(text string, _ int) {
		manager.selectedDB = text
		if err := manager.loadSuperusers(ui.dispatcher); err != nil {
			ui.showError(err)
			return
		}
		fillSuperuserTable()
	})
	dbForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			closeFn()
			return nil
		}
		return remapFormArrowNavigation(currentFormPrimitive(dbForm), event)
	})

	hint := tview.NewTextView().SetText("[n]ew  [e]dit  [D]elete  [Esc]close").SetTextAlign(tview.AlignCenter)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true).SetTitle(" Superusers ")
	container.AddItem(dbForm, 3, 0, false)
	container.AddItem(table, 0, 1, true)
	container.AddItem(hint, 1, 0, false)

	ui.modalOpen = true
	ui.pages.AddPage(pageName, center(80, 16, container), true, true)
	ui.app.SetFocus(table)
}

func (ui *navigatorTUI) openSuperuserEditModal(dbAlias string, item *storage.Superuser, onBack func()) {
	const pageName = "superuser-edit"
	isNew := item == nil

	dbs, err := ui.dispatcher.dbStore.List()
	if err != nil {
		ui.showError(mapStoreError(err))
		return
	}
	manager := newSuperuserManagerState(dbs, dbAlias)
	if err := manager.loadSuperusers(ui.dispatcher); err != nil {
		ui.showError(err)
		return
	}

	var title, aliasValue, emailValue, submitLabel string
	if isNew {
		title = " New Superuser "
		submitLabel = "Add"
	} else {
		manager.selectAlias(item.Alias)
		title = " Edit: " + item.Alias + " "
		aliasValue = item.Alias
		emailValue = item.Email
		submitLabel = "Update"
	}

	backFn := func() {
		ui.pages.RemovePage(pageName)
		onBack()
	}

	form := tview.NewForm()
	form.AddInputField("alias", aliasValue, 0, nil, nil)
	form.AddInputField("email", emailValue, 0, nil, nil)
	form.AddPasswordField("password(blank=keep)", "", 0, '*', nil)

	aliasField := form.GetFormItem(0).(*tview.InputField)
	emailField := form.GetFormItem(1).(*tview.InputField)
	passwordField := form.GetFormItem(2).(*tview.InputField)

	form.AddButton(submitLabel, func() {
		ui.saveSuperuserManager(manager, aliasField.GetText(), emailField.GetText(), passwordField.GetText())
	})
	form.AddButton("Back", backFn)
	form.SetBorder(true).SetTitle(title)
	form.SetButtonsAlign(tview.AlignRight)
	installFormArrowNavigationWithClose(form, backFn)

	ui.modalOpen = true
	ui.pages.AddPage(pageName, center(80, 14, form), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) saveDBManager(manager dbManagerState, alias, baseURL string) {
	previousAlias := manager.selectedAlias
	ui.pages.RemovePage("db-edit")
	ui.closeModal("db-list")
	if err := manager.save(ui.dispatcher, alias, baseURL); err != nil {
		ui.showError(err)
		return
	}

	ui.updateSessionDB(previousAlias, alias)
	ui.reloadAfterLocalConfigChange("db aliases updated")
}

func (ui *navigatorTUI) deleteDBManager(manager dbManagerState, onCancel func()) {
	ui.closeModal("db-list")
	ui.openConfirmModal("Delete alias '"+manager.selectedAlias+"'?", func() {
		status := dbDeleteStatus(ui.session.DB.Alias, manager.selectedAlias)
		if err := manager.remove(ui.dispatcher); err != nil {
			ui.showError(err)
			return
		}
		ui.reloadAfterLocalConfigChange(status)
	}, onCancel)
}

func (ui *navigatorTUI) saveSuperuserManager(manager superuserManagerState, alias, email, password string) {
	previousAlias := manager.selectedAlias
	ui.pages.RemovePage("superuser-edit")
	ui.closeModal("superuser-list")
	if err := manager.save(ui.dispatcher, alias, email, password); err != nil {
		ui.showError(err)
		return
	}

	ui.updateSessionSuperuser(manager.selectedDB, previousAlias, alias)
	ui.reloadAfterLocalConfigChange("superusers updated")
}

func (ui *navigatorTUI) deleteSuperuserManager(manager superuserManagerState, onCancel func()) {
	ui.closeModal("superuser-list")
	ui.openConfirmModal("Delete superuser '"+manager.selectedAlias+"'?", func() {
		status := superuserDeleteStatus(ui.session, manager.selectedDB, manager.selectedAlias)
		if err := manager.remove(ui.dispatcher); err != nil {
			ui.showError(err)
			return
		}
		ui.reloadAfterLocalConfigChange(status)
	}, onCancel)
}

func (ui *navigatorTUI) reloadAfterLocalConfigChange(status string) {
	if err := ui.syncLocalConfigState(); err != nil {
		ui.showError(err)
		return
	}

	ui.statusMessage = status
	ui.renderCurrentScreen()
}

func (ui *navigatorTUI) closeModal(name string) {
	ui.pages.RemovePage(name)
	ui.modalOpen = false
	ui.focusMain()
}

func (ui *navigatorTUI) showError(err error) {
	ui.modalOpen = true
	body := err.Error()
	if formatted := apperr.Format(err); strings.TrimSpace(formatted) != "" {
		body = formatted
	}
	text := tview.NewTextView().SetText(body)
	text.SetBorder(true).SetTitle(" Error ")

	form := tview.NewForm().AddButton("OK", func() {
		ui.dismissErrorModal()
	})
	form.SetButtonsAlign(tview.AlignCenter)
	installFormArrowNavigationWithClose(form, ui.dismissErrorModal)
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.AddItem(text, 0, 1, false)
	container.AddItem(form, 3, 0, true)

	ui.pages.AddPage("error", center(80, 12, container), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) focusMain() {
	if ui.modalOpen || ui.app == nil {
		return
	}
	if ui.isRecordDetailScreen() {
		ui.app.SetFocus(ui.detailView)
		return
	}
	if ui.tableView != nil {
		ui.app.SetFocus(ui.tableView)
	}
}

func (ui *navigatorTUI) openConfirmModal(message string, onConfirm func(), onCancel func()) {
	const pageName = "confirm"
	form := tview.NewForm()
	form.AddButton("Confirm", func() {
		ui.closeModal(pageName)
		onConfirm()
	})
	form.AddButton("Cancel", func() {
		ui.closeModal(pageName)
		if onCancel != nil {
			onCancel()
		}
	})
	form.SetButtonsAlign(tview.AlignCenter)

	fullMessage := message + "\n\n[Enter] confirm   [Esc] cancel"
	text := tview.NewTextView().SetText(fullMessage).SetTextAlign(tview.AlignCenter)
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.AddItem(text, 0, 1, false)
	container.AddItem(form, 1, 0, true)

	installFormArrowNavigationWithClose(form, func() {
		ui.closeModal(pageName)
		if onCancel != nil {
			onCancel()
		}
	})
	ui.modalOpen = true
	ui.pages.AddPage(pageName, center(60, 7, container), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) dismissErrorModal() {
	ui.pages.RemovePage("error")
	ui.modalOpen = false
	ui.focusMain()
}

func (ui *navigatorTUI) setLoadingStatus() {
	ui.statusMessage = "loading…"
	ui.statusView.SetText(ui.statusText())
}

func (ui *navigatorTUI) clearLoadingStatus() {
	ui.statusMessage = ""
	if ui.statusView != nil {
		ui.statusView.SetText(ui.statusText())
	}
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

func buildManagerTable(rows [][]string) *tview.Table {
	table := tview.NewTable().SetSelectable(true, false)
	table.SetBorderPadding(0, 0, 1, 1)
	if len(rows) == 0 {
		table.SetCell(0, 0, tview.NewTableCell("No entries yet. Press New to add one."))
		table.SetSelectable(false, false)
		return table
	}
	for r, row := range rows {
		for c, text := range row {
			cell := tview.NewTableCell(text).SetExpansion(1).SetMaxWidth(0)
			if c == 0 {
				cell.SetAttributes(tcell.AttrBold)
			}
			table.SetCell(r, c, cell)
		}
	}
	return table
}

func dbListRows(items []storage.DB) [][]string {
	rows := make([][]string, len(items))
	for i, item := range items {
		rows[i] = []string{item.Alias, item.BaseURL}
	}
	return rows
}

func center(width, height int, primitive tview.Primitive) tview.Primitive {
	row := tview.NewFlex().SetDirection(tview.FlexRow)
	row.AddItem(nil, 0, 1, false)
	row.AddItem(primitive, height, 1, true)
	row.AddItem(nil, 0, 1, false)

	col := tview.NewFlex()
	col.AddItem(nil, 0, 1, false)
	col.AddItem(row, width, 1, true)
	col.AddItem(nil, 0, 1, false)
	return col
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

func collectionName(row map[string]any) string {
	if row == nil {
		return ""
	}
	if name := strings.TrimSpace(formatValue(row["name"])); name != "" {
		return name
	}
	return strings.TrimSpace(formatValue(row["id"]))
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

func installFormArrowNavigation(form *tview.Form) {
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return remapFormArrowNavigation(currentFormPrimitive(form), event)
	})
}

func installFormArrowNavigationWithClose(form *tview.Form, onClose func()) {
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event != nil && event.Key() == tcell.KeyEsc {
			if onClose != nil {
				onClose()
			}
			return nil
		}
		return remapFormArrowNavigation(currentFormPrimitive(form), event)
	})
}

func installSubmitCancelNavigation(form *tview.Form, apply, cancel func()) {
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return remapSubmitCancelNavigation(currentFormPrimitive(form), event, apply, cancel)
	})
}

func currentFormPrimitive(form *tview.Form) tview.Primitive {
	if form == nil {
		return nil
	}
	if itemIndex, buttonIndex := form.GetFocusedItemIndex(); itemIndex >= 0 {
		return form.GetFormItem(itemIndex)
	} else if buttonIndex >= 0 {
		return form.GetButton(buttonIndex)
	}
	return nil
}

func remapFormArrowNavigation(focused tview.Primitive, event *tcell.EventKey) *tcell.EventKey {
	if event == nil {
		return nil
	}

	if key, ok := formNavigationKey(focused, event.Key()); ok {
		return tcell.NewEventKey(key, 0, tcell.ModNone)
	}
	return event
}

func remapSubmitCancelNavigation(focused tview.Primitive, event *tcell.EventKey, apply, cancel func()) *tcell.EventKey {
	if event == nil {
		return nil
	}

	switch event.Key() {
	case tcell.KeyEnter:
		if apply != nil {
			apply()
		}
		return nil
	case tcell.KeyEsc:
		if cancel != nil {
			cancel()
		}
		return nil
	default:
		return remapFormArrowNavigation(focused, event)
	}
}

func formNavigationKey(focused tview.Primitive, key tcell.Key) (tcell.Key, bool) {
	if isOpenDropDown(focused) {
		return 0, false
	}

	switch focused.(type) {
	case *tview.InputField:
		switch key {
		case tcell.KeyUp:
			return tcell.KeyBacktab, true
		case tcell.KeyDown:
			return tcell.KeyTab, true
		default:
			return 0, false
		}
	default:
		switch key {
		case tcell.KeyUp, tcell.KeyLeft:
			return tcell.KeyBacktab, true
		case tcell.KeyDown, tcell.KeyRight:
			return tcell.KeyTab, true
		default:
			return 0, false
		}
	}
}

func isOpenDropDown(focused tview.Primitive) bool {
	dropdown, ok := focused.(*tview.DropDown)
	return ok && dropdown.IsOpen()
}

func mergeColumns(observed map[string]struct{}, fresh []string) []string {
	for _, col := range fresh {
		observed[col] = struct{}{}
	}
	cols := make([]string, 0, len(observed))
	for col := range observed {
		cols = append(cols, col)
	}
	sort.Slice(cols, func(i, j int) bool {
		return cols[i] < cols[j]
	})
	return cols
}

func dbAliasOptions(items []storage.DB) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Alias)
	}
	return out
}

func superuserAliasOptions(items []storage.Superuser) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Alias)
	}
	return out
}

func newSuperuserManagerState(dbs []storage.DB, currentDB string) superuserManagerState {
	selectedDB := currentDB
	if indexDBAlias(dbs, currentDB) < 0 {
		selectedDB = dbs[0].Alias
	}
	return superuserManagerState{dbs: dbs, selectedDB: selectedDB}
}

func (m *superuserManagerState) loadSuperusers(dispatcher *Dispatcher) error {
	items, err := dispatcher.suStore.ListByDB(m.selectedDB)
	if err != nil {
		return mapStoreError(err)
	}
	m.superusers = items
	m.selectedAlias = ""
	return nil
}

func (m superuserManagerState) selectedDBIndex() int {
	index := indexDBAlias(m.dbs, m.selectedDB)
	if index < 0 {
		return 0
	}
	return index
}

func normalizeManagerSelection(value string) string {
	if value == managerNewOption {
		return ""
	}
	return strings.TrimSpace(value)
}

func findDB(items []storage.DB, alias string) (storage.DB, bool) {
	for _, item := range items {
		if strings.EqualFold(item.Alias, alias) {
			return item, true
		}
	}
	return storage.DB{}, false
}

func findSuperuser(items []storage.Superuser, alias string) (storage.Superuser, bool) {
	for _, item := range items {
		if strings.EqualFold(item.Alias, alias) {
			return item, true
		}
	}
	return storage.Superuser{}, false
}

func indexDBAlias(items []storage.DB, alias string) int {
	for i, item := range items {
		if strings.EqualFold(item.Alias, alias) {
			return i
		}
	}
	return -1
}

func dbDeleteStatus(currentTargetAlias, deletedAlias string) string {
	if strings.EqualFold(currentTargetAlias, deletedAlias) {
		return "current db alias was removed from local config"
	}
	return "db aliases updated"
}

func superuserDeleteStatus(session pbSession, deletedDBAlias, deletedAlias string) string {
	if strings.EqualFold(session.DB.Alias, deletedDBAlias) && strings.EqualFold(session.SU.Alias, deletedAlias) {
		return "current superuser alias was removed from local config"
	}
	return "superusers updated"
}
