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
)

type navigatorRoute struct {
	screen    navigatorScreen
	target    pbTarget
	hasTarget bool
	state     RecordsQueryState
}

type navigatorTUI struct {
	dispatcher *Dispatcher
	ctx        context.Context

	app    *tview.Application
	pages  *tview.Pages
	layout *tview.Flex

	statusView *tview.TextView
	tableView  *tview.Table
	detailView *tview.TextView
	helpView   *tview.TextView

	screen      navigatorScreen
	history     []navigatorScreen
	modalOpen   bool
	hasTarget   bool
	target      pbTarget
	dbs         []storage.DB
	superusers  []storage.Superuser
	collections []map[string]any

	recordsState  RecordsQueryState
	result        pocketbase.QueryResult
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

func (m dbManagerState) choices() []string {
	return dbManagerChoices(m.items)
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
		return apperr.Invalid("Select an existing db alias first.", "Choose a saved db alias from the target field.")
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
		return apperr.Invalid("Select an existing superuser first.", "Choose a saved superuser from the target field.")
	}

	return dispatcher.removeSuperuser(m.selectedDB, m.selectedAlias)
}

func (d *Dispatcher) RunRecordsTUI(ctx context.Context, target pbTarget, state RecordsQueryState) error {
	return d.runRecordsTUI(ctx, target, state)
}

func (d *Dispatcher) runRecordsTUI(ctx context.Context, target pbTarget, state RecordsQueryState) error {
	return d.navigatorRunner(ctx, navigatorRoute{
		screen:    screenRecords,
		target:    target,
		hasTarget: true,
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

	err := ui.app.SetRoot(ui.pages, true).SetFocus(ui.tableView).Run()
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
	if route.hasTarget && route.screen == screenRecords {
		ui.hasTarget = true
		ui.target = route.target
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
	ui.app.SetInputCapture(ui.handleKey)
	ui.statusView.SetDynamicColors(true)

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
		ui.renderDetail()
	})

	ui.detailView.SetTextAlign(tview.AlignLeft)
	ui.detailView.SetDynamicColors(true)
	ui.detailView.SetBorder(true)

	ui.layout = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.layout.AddItem(ui.statusView, 1, 0, false)
	ui.layout.AddItem(ui.tableView, 0, 1, true)
	ui.layout.AddItem(ui.detailView, 9, 0, false)
	ui.layout.AddItem(ui.helpView, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", ui.layout, true, true)
	ui.pages.SetInputCapture(ui.handleKey)

	ui.renderCurrentScreen()
}

func (ui *navigatorTUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.modalOpen {
		return event
	}

	switch event.Key() {
	case tcell.KeyEsc, tcell.KeyBackspace, tcell.KeyBackspace2:
		ui.goBack()
		return nil
	case tcell.KeyEnter:
		ui.handleEnter()
		return nil
	case tcell.KeyLeft:
		if ui.screen == screenRecords {
			ui.shiftColumns(-1)
			return nil
		}
	case tcell.KeyRight:
		if ui.screen == screenRecords {
			ui.shiftColumns(1)
			return nil
		}
	}

	switch event.Rune() {
	case 'q':
		ui.app.Stop()
		return nil
	case 'j':
		ui.moveSelection(1)
		return nil
	case 'k':
		ui.moveSelection(-1)
		return nil
	case 'h':
		if ui.screen == screenRecords {
			ui.shiftColumns(-1)
			return nil
		}
	case 'l':
		if ui.screen == screenRecords {
			ui.shiftColumns(1)
			return nil
		}
	case '/':
		if ui.screen == screenRecords {
			ui.openInputModal("Filter", "filter", ui.recordsState.Filter, func(val string) error {
				ui.recordsState.Filter = strings.TrimSpace(val)
				ui.recordsState.Page = 1
				return ui.fetchAndRenderRecords()
			})
			return nil
		}
	case 's':
		if ui.screen == screenRecords {
			ui.openInputModal("Sort", "sort", ui.recordsState.Sort, func(val string) error {
				ui.recordsState.Sort = strings.TrimSpace(val)
				ui.recordsState.Page = 1
				return ui.fetchAndRenderRecords()
			})
			return nil
		}
	case 'c':
		if ui.screen == screenRecords {
			ui.openColumnsModal()
			return nil
		}
	case 'b':
		ui.openDBManagerModal()
		return nil
	case 'u':
		ui.openSuperuserManagerModal()
		return nil
	case 'r':
		_ = ui.refreshCurrentScreen()
		return nil
	case '[':
		if ui.screen == screenRecords && ui.recordsState.Page > 1 {
			ui.recordsState.Page--
			_ = ui.fetchAndRenderRecords()
		}
		return nil
	case ']':
		if ui.screen == screenRecords && (ui.totalPages == 0 || ui.recordsState.Page < ui.totalPages) {
			ui.recordsState.Page++
			_ = ui.fetchAndRenderRecords()
		}
		return nil
	case 'g':
		if ui.screen == screenRecords {
			ui.recordsState.Page = 1
			_ = ui.fetchAndRenderRecords()
		}
		return nil
	case 'G':
		if ui.screen == screenRecords && ui.totalPages > 0 {
			ui.recordsState.Page = ui.totalPages
			_ = ui.fetchAndRenderRecords()
		}
		return nil
	}
	return event
}

func (ui *navigatorTUI) handleEnter() {
	switch ui.screen {
	case screenDBList:
		if err := ui.activateSelectedDB(); err != nil {
			ui.showError(err)
		}
	case screenSuperusers:
		if err := ui.activateSelectedSuperuser(); err != nil {
			ui.showError(err)
		}
	case screenCollections:
		if err := ui.activateSelectedCollection(); err != nil {
			ui.showError(err)
		}
	case screenRecords:
		ui.toggleDetail()
	}
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
		if err := ui.loadDBs(); err != nil {
			ui.showError(err)
			return err
		}
	case screenSuperusers:
		if err := ui.loadSuperusers(); err != nil {
			ui.showError(err)
			return err
		}
	case screenCollections:
		if err := ui.loadCollections(); err != nil {
			ui.showError(err)
			return err
		}
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

	ui.hasTarget = true
	ui.target = pbTarget{DB: db}
	ui.dispatcher.sessionCtx.DBAlias = db.Alias
	ui.dispatcher.sessionCtx.SuperuserAlias = ""

	if preferred, ok := pickPreferredSuperuser(db.Alias, superusers, previousSession, ui.dispatcher.savedCtx); ok {
		ui.target.SU = preferred
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
	ui.target.SU = su
	ui.dispatcher.sessionCtx = commandContext{DBAlias: ui.target.DB.Alias, SuperuserAlias: su.Alias}
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
	ui.detailVisible = true
	ui.observedCols = map[string]struct{}{}
	if err := ui.fetchRecords(); err != nil {
		return err
	}
	ui.pushScreen(screenRecords)
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
	if !ui.hasTarget {
		return apperr.RuntimeErr("No db is selected.", "", nil)
	}
	items, err := ui.dispatcher.suStore.ListByDB(ui.target.DB.Alias)
	if err != nil {
		return mapStoreError(err)
	}
	ui.superusers = items
	return nil
}

func (ui *navigatorTUI) loadCollections() error {
	if !ui.hasTarget {
		return apperr.RuntimeErr("No target is selected.", "", nil)
	}
	result, err := ui.dispatcher.fetchCollections(ui.ctx, ui.target)
	if err != nil {
		return err
	}
	ui.collections = result.Rows
	return nil
}

func (d *Dispatcher) fetchCollections(ctx context.Context, target pbTarget) (pocketbase.QueryResult, error) {
	payload, err := d.getJSONWithAuth(ctx, target, pocketbase.BuildCollectionsEndpoint(), nil)
	if err != nil {
		return pocketbase.QueryResult{}, err
	}
	return pocketbase.ParseItemsResult(payload), nil
}

func (ui *navigatorTUI) fetchRecords() error {
	result, err := ui.dispatcher.fetchRecords(ui.ctx, ui.target, ui.recordsState)
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
	if err := ui.fetchRecords(); err != nil {
		ui.showError(err)
		return err
	}
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
		cols := pocketbase.CollectColumns(ui.collectionRows())
		if len(cols) == 0 {
			return []string{"result"}
		}
		return cols
	case screenRecords:
		if len(ui.recordsState.Fields) > 0 {
			return ui.recordsState.Fields
		}
		cols := pocketbase.CollectColumns(ui.result.Rows)
		if len(cols) == 0 {
			return []string{"result"}
		}
		return cols
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
	return ui.collections
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
	if ui.screen != screenRecords || ui.detailVisible {
		ui.layout.ResizeItem(ui.detailView, 9, 0)
	} else {
		ui.layout.ResizeItem(ui.detailView, 0, 0)
	}
	ui.statusView.SetText(ui.statusText())
	ui.helpView.SetText(ui.helpText())
	ui.detailView.SetTitle(" " + ui.detailTitle() + " ")
	ui.renderTable()
	ui.renderDetail()
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

func (ui *navigatorTUI) emptyDetailText() string {
	switch ui.screen {
	case screenDBList:
		return "No saved db aliases"
	case screenSuperusers:
		return "No superusers"
	case screenCollections:
		return "No collections"
	case screenRecords:
		return "No records"
	default:
		return "No data"
	}
}

func (ui *navigatorTUI) moveSelection(delta int) {
	rows := ui.currentRows()
	if len(rows) == 0 {
		return
	}
	next := ui.selectedIndex + delta
	if next < 0 {
		next = 0
	}
	if next >= len(rows) {
		next = len(rows) - 1
	}
	ui.selectedIndex = next
	ui.tableView.Select(ui.selectedIndex+1, 0)
	ui.renderDetail()
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

func (ui *navigatorTUI) toggleDetail() {
	ui.detailVisible = !ui.detailVisible
	if ui.detailVisible {
		ui.layout.ResizeItem(ui.detailView, 9, 0)
	} else {
		ui.layout.ResizeItem(ui.detailView, 0, 0)
	}
}

func (ui *navigatorTUI) statusText() string {
	parts := []string{"path=" + ui.breadcrumb()}
	if ui.hasTarget {
		parts = append(parts, "db="+ui.target.DB.Alias)
		if strings.TrimSpace(ui.target.SU.Alias) != "" {
			parts = append(parts, "superuser="+ui.target.SU.Alias)
		}
	}
	if ui.screen == screenRecords {
		parts = append(parts,
			fmt.Sprintf("collection=%s", ui.recordsState.Collection),
			fmt.Sprintf("page=%d", ui.recordsState.Page),
			fmt.Sprintf("perPage=%d", ui.recordsState.PerPage),
			fmt.Sprintf("totalItems=%d", ui.totalItems),
			fmt.Sprintf("totalPages=%d", ui.totalPages),
		)
		if strings.TrimSpace(ui.recordsState.Filter) != "" {
			parts = append(parts, fmt.Sprintf("filter=%q", ui.recordsState.Filter))
		}
		if strings.TrimSpace(ui.recordsState.Sort) != "" {
			parts = append(parts, fmt.Sprintf("sort=%q", ui.recordsState.Sort))
		}
	}
	if strings.TrimSpace(ui.statusMessage) != "" {
		parts = append(parts, "status="+ui.statusMessage)
	}
	return strings.Join(parts, "  ")
}

func (ui *navigatorTUI) breadcrumb() string {
	trail := []string{"dbs"}
	if ui.hasTarget {
		trail = append(trail, ui.target.DB.Alias)
	}
	if ui.screen == screenSuperusers || ui.screen == screenCollections || ui.screen == screenRecords {
		if strings.TrimSpace(ui.target.SU.Alias) != "" {
			trail = append(trail, ui.target.SU.Alias)
		} else if ui.screen == screenSuperusers {
			trail = append(trail, "superusers")
		}
	}
	if ui.screen == screenCollections || ui.screen == screenRecords {
		trail = append(trail, "collections")
	}
	if ui.screen == screenRecords && strings.TrimSpace(ui.recordsState.Collection) != "" {
		trail = append(trail, ui.recordsState.Collection)
	}
	return strings.Join(trail, " > ")
}

func (ui *navigatorTUI) helpText() string {
	switch ui.screen {
	case screenDBList:
		return "q quit  j/k move  Enter select  b db aliases  u superusers  r refresh"
	case screenSuperusers:
		return "q quit  esc/backspace back  j/k move  Enter select  b db aliases  u superusers  r refresh"
	case screenCollections:
		return "q quit  esc/backspace back  j/k move  Enter select  b db aliases  u superusers  r refresh"
	case screenRecords:
		return "q quit  esc/backspace back  j/k move  h/l or <-/-> horiz  / filter  s sort  c columns  b db aliases  u superusers  [/] page  g/G first/last  r refresh  Enter detail"
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
	default:
		return "detail"
	}
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

	modal := center(60, 9, form)
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
	form.AddButton("Apply", func() {
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
	})
	form.AddButton("Clear", func() {
		ui.recordsState.Fields = nil
		ui.recordsState.Page = 1
		ui.closeModal("columns")
		_ = ui.fetchAndRenderRecords()
	})
	form.AddButton("Cancel", func() {
		ui.closeModal("columns")
	})
	form.SetBorder(true).SetTitle(" Columns ")

	height := len(cols) + 6
	if height > 24 {
		height = 24
	}
	modal := center(70, height, form)
	ui.pages.AddPage("columns", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) openDBManagerModal() {
	items, err := ui.dispatcher.dbStore.List()
	if err != nil {
		ui.showError(mapStoreError(err))
		return
	}

	manager := newDBManagerState(items)
	form := tview.NewForm()
	form.AddDropDown("target", manager.choices(), 0, nil)
	form.AddInputField("alias", "", 0, nil, nil)
	form.AddInputField("base url", "", 0, nil, nil)

	dropdown := form.GetFormItem(0).(*tview.DropDown)
	aliasField := form.GetFormItem(1).(*tview.InputField)
	baseURLField := form.GetFormItem(2).(*tview.InputField)

	dropdown.SetSelectedFunc(func(text string, _ int) {
		applyDBFormSelection(&manager, aliasField, baseURLField, text)
	})
	applyDBFormSelection(&manager, aliasField, baseURLField, managerNewOption)

	form.AddButton("Save", func() {
		ui.saveDBManager(manager, aliasField.GetText(), baseURLField.GetText())
	})
	form.AddButton("Delete", func() {
		ui.deleteDBManager(manager)
	})
	form.AddButton("Close", func() {
		ui.closeModal("db-manager")
	})
	form.SetBorder(true).SetTitle(" DB Aliases ")
	form.SetButtonsAlign(tview.AlignRight)

	ui.modalOpen = true
	ui.pages.AddPage("db-manager", center(76, 12, form), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) openSuperuserManagerModal() {
	dbs, err := ui.dispatcher.dbStore.List()
	if err != nil {
		ui.showError(mapStoreError(err))
		return
	}
	if len(dbs) == 0 {
		ui.showError(apperr.Invalid("No db aliases are configured.", "Save a db alias before adding superusers."))
		return
	}

	manager := newSuperuserManagerState(dbs, ui.target.DB.Alias)
	if err := manager.loadSuperusers(ui.dispatcher); err != nil {
		ui.showError(err)
		return
	}

	form := tview.NewForm()
	form.AddDropDown("db", dbAliasOptions(dbs), manager.selectedDBIndex(), nil)
	form.AddDropDown("target", manager.aliasChoices(), 0, nil)
	form.AddInputField("alias", "", 0, nil, nil)
	form.AddInputField("email", "", 0, nil, nil)
	form.AddPasswordField("password(blank=keep)", "", 0, '*', nil)

	dbDropdown := form.GetFormItem(0).(*tview.DropDown)
	targetDropdown := form.GetFormItem(1).(*tview.DropDown)
	aliasField := form.GetFormItem(2).(*tview.InputField)
	emailField := form.GetFormItem(3).(*tview.InputField)
	passwordField := form.GetFormItem(4).(*tview.InputField)

	dbDropdown.SetSelectedFunc(func(text string, _ int) {
		manager.selectedDB = text
		if err := manager.loadSuperusers(ui.dispatcher); err != nil {
			ui.showError(err)
			return
		}
		targetDropdown.SetOptions(manager.aliasChoices(), func(option string, _ int) {
			applySuperuserFormSelection(&manager, aliasField, emailField, passwordField, option)
		})
		targetDropdown.SetCurrentOption(0)
		applySuperuserFormSelection(&manager, aliasField, emailField, passwordField, managerNewOption)
	})
	targetDropdown.SetSelectedFunc(func(text string, _ int) {
		applySuperuserFormSelection(&manager, aliasField, emailField, passwordField, text)
	})
	applySuperuserFormSelection(&manager, aliasField, emailField, passwordField, managerNewOption)

	form.AddButton("Save", func() {
		ui.saveSuperuserManager(manager, aliasField.GetText(), emailField.GetText(), passwordField.GetText())
	})
	form.AddButton("Delete", func() {
		ui.deleteSuperuserManager(manager)
	})
	form.AddButton("Close", func() {
		ui.closeModal("superuser-manager")
	})
	form.SetBorder(true).SetTitle(" Superusers ")
	form.SetButtonsAlign(tview.AlignRight)

	ui.modalOpen = true
	ui.pages.AddPage("superuser-manager", center(80, 14, form), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) saveDBManager(manager dbManagerState, alias, baseURL string) {
	previousAlias := manager.selectedAlias
	ui.closeModal("db-manager")
	if err := manager.save(ui.dispatcher, alias, baseURL); err != nil {
		ui.showError(err)
		return
	}

	ui.retargetDBAlias(previousAlias, alias)
	ui.reloadAfterLocalConfigChange("db aliases updated")
}

func (ui *navigatorTUI) deleteDBManager(manager dbManagerState) {
	status := dbDeleteStatus(ui.target.DB.Alias, manager.selectedAlias)
	ui.closeModal("db-manager")
	if err := manager.remove(ui.dispatcher); err != nil {
		ui.showError(err)
		return
	}

	ui.reloadAfterLocalConfigChange(status)
}

func (ui *navigatorTUI) saveSuperuserManager(manager superuserManagerState, alias, email, password string) {
	previousAlias := manager.selectedAlias
	ui.closeModal("superuser-manager")
	if err := manager.save(ui.dispatcher, alias, email, password); err != nil {
		ui.showError(err)
		return
	}

	ui.retargetSuperuserAlias(manager.selectedDB, previousAlias, alias)
	ui.reloadAfterLocalConfigChange("superusers updated")
}

func (ui *navigatorTUI) deleteSuperuserManager(manager superuserManagerState) {
	status := superuserDeleteStatus(ui.target, manager.selectedDB, manager.selectedAlias)
	ui.closeModal("superuser-manager")
	if err := manager.remove(ui.dispatcher); err != nil {
		ui.showError(err)
		return
	}

	ui.reloadAfterLocalConfigChange(status)
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
	ui.app.SetFocus(ui.tableView)
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
		ui.pages.RemovePage("error")
		ui.modalOpen = false
		ui.app.SetFocus(ui.tableView)
	})
	form.SetButtonsAlign(tview.AlignCenter)
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.AddItem(text, 0, 1, false)
	container.AddItem(form, 3, 0, true)

	ui.pages.AddPage("error", center(80, 12, container), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) syncLocalConfigState() error {
	if err := ui.loadDBs(); err != nil {
		return err
	}
	if !ui.hasTarget {
		return nil
	}

	db, found, err := ui.dispatcher.dbStore.Find(ui.target.DB.Alias)
	if err != nil {
		return mapStoreError(err)
	}
	if !found {
		ui.resetToDBList()
		return nil
	}
	ui.target.DB = db

	if strings.TrimSpace(ui.target.SU.Alias) != "" {
		su, found, err := ui.dispatcher.suStore.Find(db.Alias, ui.target.SU.Alias)
		if err != nil {
			return mapStoreError(err)
		}
		if found {
			ui.target.SU = su
		} else {
			ui.target.SU = storage.Superuser{}
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
		if strings.TrimSpace(ui.target.SU.Alias) == "" {
			ui.screen = screenSuperusers
			return ui.loadSuperusers()
		}
		return ui.loadCollections()
	case screenRecords:
		if strings.TrimSpace(ui.target.SU.Alias) == "" {
			ui.screen = screenSuperusers
			return ui.loadSuperusers()
		}
		if err := ui.loadCollections(); err != nil {
			return err
		}
		return ui.fetchRecords()
	default:
		return nil
	}
}

func (ui *navigatorTUI) resetToDBList() {
	ui.screen = screenDBList
	ui.history = nil
	ui.hasTarget = false
	ui.target = pbTarget{}
	ui.superusers = nil
	ui.collections = nil
	ui.result = pocketbase.QueryResult{}
	ui.totalItems = 0
	ui.totalPages = 0
	ui.selectedIndex = 0
	ui.columnOffset = 0
}

func (ui *navigatorTUI) retargetDBAlias(previousAlias, nextAlias string) {
	if !ui.hasTarget || strings.TrimSpace(previousAlias) == "" {
		return
	}
	if !strings.EqualFold(ui.target.DB.Alias, previousAlias) {
		return
	}

	ui.target.DB.Alias = nextAlias
	if strings.EqualFold(ui.target.SU.DBAlias, previousAlias) {
		ui.target.SU.DBAlias = nextAlias
	}
}

func (ui *navigatorTUI) retargetSuperuserAlias(dbAlias, previousAlias, nextAlias string) {
	if !ui.hasTarget || strings.TrimSpace(previousAlias) == "" {
		return
	}
	if !strings.EqualFold(ui.target.DB.Alias, dbAlias) {
		return
	}
	if !strings.EqualFold(ui.target.SU.Alias, previousAlias) {
		return
	}

	ui.target.SU.Alias = nextAlias
}

func applyDBFormSelection(manager *dbManagerState, aliasField, baseURLField *tview.InputField, value string) {
	db, ok := manager.selectAlias(value)
	if !ok {
		aliasField.SetText("")
		baseURLField.SetText("")
		return
	}

	aliasField.SetText(db.Alias)
	baseURLField.SetText(db.BaseURL)
}

func applySuperuserFormSelection(manager *superuserManagerState, aliasField, emailField, passwordField *tview.InputField, value string) {
	su, ok := manager.selectAlias(value)
	if !ok {
		aliasField.SetText("")
		emailField.SetText("")
		passwordField.SetText("")
		return
	}

	aliasField.SetText(su.Alias)
	emailField.SetText(su.Email)
	passwordField.SetText("")
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

func dbManagerChoices(items []storage.DB) []string {
	return append([]string{managerNewOption}, dbAliasOptions(items)...)
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

func (m superuserManagerState) aliasChoices() []string {
	return append([]string{managerNewOption}, superuserAliasOptions(m.superusers)...)
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

func superuserDeleteStatus(target pbTarget, deletedDBAlias, deletedAlias string) string {
	if strings.EqualFold(target.DB.Alias, deletedDBAlias) && strings.EqualFold(target.SU.Alias, deletedAlias) {
		return "current superuser alias was removed from local config"
	}
	return "superusers updated"
}
