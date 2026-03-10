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

type recordsTUI struct {
	dispatcher *Dispatcher
	target     pbTarget
	state      RecordsQueryState
	ctx        context.Context

	app    *tview.Application
	pages  *tview.Pages
	layout *tview.Flex

	statusView *tview.TextView
	tableView  *tview.Table
	detailView *tview.TextView
	helpView   *tview.TextView

	result        pocketbase.QueryResult
	totalItems    int
	totalPages    int
	selectedIndex int
	columnOffset  int
	detailVisible bool
	modalOpen     bool
	observedCols  map[string]struct{}
	statusMessage string
}

type dbManagerState struct {
	items          []storage.DB
	selectedAlias  string
	selectedRecord storage.DB
}

type superuserManagerState struct {
	dbs            []storage.DB
	selectedDB     string
	superusers     []storage.Superuser
	selectedAlias  string
	selectedRecord storage.Superuser
}

func (d *Dispatcher) runRecordsTUI(ctx context.Context, target pbTarget, state RecordsQueryState) error {
	ui := newRecordsTUI(d, ctx, target, state)
	if err := ui.fetch(); err != nil {
		return err
	}
	ui.setupViews()
	return ui.run()
}

func newRecordsTUI(dispatcher *Dispatcher, ctx context.Context, target pbTarget, state RecordsQueryState) *recordsTUI {
	return &recordsTUI{
		dispatcher:    dispatcher,
		target:        target,
		state:         state,
		ctx:           ctx,
		app:           tview.NewApplication(),
		statusView:    tview.NewTextView(),
		tableView:     tview.NewTable(),
		detailView:    tview.NewTextView(),
		helpView:      tview.NewTextView(),
		detailVisible: true,
		observedCols:  map[string]struct{}{},
	}
}

func (ui *recordsTUI) run() error {
	done := make(chan struct{})
	go func() {
		select {
		case <-ui.ctx.Done():
			ui.app.Stop()
		case <-done:
		}
	}()

	err := ui.app.SetRoot(ui.pages, true).SetFocus(ui.tableView).Run()
	close(done)
	if err != nil {
		return apperr.RuntimeErr("Could not run TUI mode.", "", err)
	}
	if ui.ctx.Err() != nil {
		return apperr.RuntimeErr("TUI mode was interrupted.", "", ui.ctx.Err())
	}
	return nil
}

func (ui *recordsTUI) setupViews() {
	ui.app.SetInputCapture(ui.handleKey)

	ui.setupStatusView()
	ui.setupTableView()
	ui.setupHelpView()
	ui.setupDetailView()
	ui.setupLayout()

	ui.render()
}

func (ui *recordsTUI) setupStatusView() {
	ui.statusView.SetDynamicColors(true)
}

func (ui *recordsTUI) setupTableView() {
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
}

func (ui *recordsTUI) setupHelpView() {
	ui.helpView.SetText("q/esc quit  j/k move  h/l or <-/-> horiz  / filter  s sort  c columns  b db aliases  u superusers  [/] page  g/G first/last  r refresh  Enter detail")
}

func (ui *recordsTUI) setupDetailView() {
	ui.detailView.SetTextAlign(tview.AlignLeft)
	ui.detailView.SetDynamicColors(true)
	ui.detailView.SetBorder(true)
	ui.detailView.SetTitle(" record detail ")
}

func (ui *recordsTUI) setupLayout() {
	ui.layout = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.layout.AddItem(ui.statusView, 1, 0, false)
	ui.layout.AddItem(ui.tableView, 0, 1, true)
	ui.layout.AddItem(ui.detailView, 9, 0, false)
	ui.layout.AddItem(ui.helpView, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", ui.layout, true, true)
	ui.applyDetailVisibility()
}

func (ui *recordsTUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.modalOpen {
		return event
	}
	if ui.handleSpecialKey(event) {
		return nil
	}
	if ui.handleRuneKey(event.Rune()) {
		return nil
	}
	return event
}

func (ui *recordsTUI) handleSpecialKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEsc:
		ui.app.Stop()
		return true
	case tcell.KeyEnter:
		ui.toggleDetail()
		return true
	case tcell.KeyLeft:
		ui.shiftColumns(-1)
		return true
	case tcell.KeyRight:
		ui.shiftColumns(1)
		return true
	default:
		return false
	}
}

func (ui *recordsTUI) handleRuneKey(key rune) bool {
	switch key {
	case 'q':
		ui.app.Stop()
	case 'j':
		ui.moveSelection(1)
	case 'k':
		ui.moveSelection(-1)
	case 'h':
		ui.shiftColumns(-1)
	case 'l':
		ui.shiftColumns(1)
	case '/':
		ui.openFilterModal()
	case 's':
		ui.openSortModal()
	case 'c':
		ui.openColumnsModal()
	case 'b':
		ui.openDBManagerModal()
	case 'u':
		ui.openSuperuserManagerModal()
	case 'r':
		_ = ui.fetchAndRender()
	case '[':
		ui.goToPreviousPage()
	case ']':
		ui.goToNextPage()
	case 'g':
		ui.goToFirstPage()
	case 'G':
		ui.goToLastPage()
	default:
		return false
	}
	return true
}

func (ui *recordsTUI) openFilterModal() {
	ui.openInputModal("Filter", "filter", ui.state.Filter, func(val string) error {
		ui.state.Filter = strings.TrimSpace(val)
		ui.state.Page = 1
		return ui.fetchAndRender()
	})
}

func (ui *recordsTUI) openSortModal() {
	ui.openInputModal("Sort", "sort", ui.state.Sort, func(val string) error {
		ui.state.Sort = strings.TrimSpace(val)
		ui.state.Page = 1
		return ui.fetchAndRender()
	})
}

func (ui *recordsTUI) goToPreviousPage() {
	if ui.state.Page <= 1 {
		return
	}
	ui.state.Page--
	_ = ui.fetchAndRender()
}

func (ui *recordsTUI) goToNextPage() {
	if ui.totalPages != 0 && ui.state.Page >= ui.totalPages {
		return
	}
	ui.state.Page++
	_ = ui.fetchAndRender()
}

func (ui *recordsTUI) goToFirstPage() {
	ui.state.Page = 1
	_ = ui.fetchAndRender()
}

func (ui *recordsTUI) goToLastPage() {
	if ui.totalPages == 0 {
		return
	}
	ui.state.Page = ui.totalPages
	_ = ui.fetchAndRender()
}

func (ui *recordsTUI) fetch() error {
	result, err := ui.dispatcher.fetchRecords(ui.ctx, ui.target, ui.state)
	if err != nil {
		return err
	}

	ui.result = result
	ui.totalItems = len(result.Rows)
	ui.totalPages = 0

	if result.Meta != nil {
		if result.Meta.Page > 0 {
			ui.state.Page = result.Meta.Page
		}
		if result.Meta.PerPage > 0 {
			ui.state.PerPage = result.Meta.PerPage
		}
		ui.totalItems = result.Meta.TotalItems
		ui.totalPages = result.Meta.TotalPages
	}
	if ui.state.Page == 0 {
		ui.state.Page = 1
	}

	ui.observeColumns()
	return nil
}

func (ui *recordsTUI) fetchAndRender() error {
	if err := ui.fetch(); err != nil {
		ui.showError(err)
		return err
	}
	ui.render()
	return nil
}

func (ui *recordsTUI) render() {
	ui.statusView.SetText(ui.statusText())
	ui.renderTable()
	ui.renderDetail()
}

func (ui *recordsTUI) renderTable() {
	ui.tableView.Clear()

	cols := ui.visibleColumns()
	for c, col := range cols {
		cell := tview.NewTableCell(strings.ToUpper(strings.ReplaceAll(col, "_", " ")))
		cell.SetSelectable(false)
		cell.SetAttributes(tcell.AttrBold)
		ui.tableView.SetCell(0, c, cell)
	}

	for r, row := range ui.result.Rows {
		for c, col := range cols {
			ui.tableView.SetCell(r+1, c, tview.NewTableCell(formatValue(row[col])))
		}
	}

	if len(ui.result.Rows) == 0 {
		ui.selectedIndex = 0
		return
	}
	if ui.selectedIndex >= len(ui.result.Rows) {
		ui.selectedIndex = len(ui.result.Rows) - 1
	}
	ui.tableView.Select(ui.selectedIndex+1, 0)
}

func (ui *recordsTUI) renderDetail() {
	if len(ui.result.Rows) == 0 {
		ui.detailView.SetText("No records")
		return
	}
	if ui.selectedIndex < 0 || ui.selectedIndex >= len(ui.result.Rows) {
		ui.selectedIndex = 0
	}

	row := ui.result.Rows[ui.selectedIndex]
	body, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		ui.detailView.SetText(formatValue(row))
		return
	}
	ui.detailView.SetText(string(body))
}

func (ui *recordsTUI) moveSelection(delta int) {
	if len(ui.result.Rows) == 0 {
		return
	}
	next := ui.selectedIndex + delta
	if next < 0 {
		next = 0
	}
	if next >= len(ui.result.Rows) {
		next = len(ui.result.Rows) - 1
	}
	ui.selectedIndex = next
	ui.tableView.Select(ui.selectedIndex+1, 0)
	ui.renderDetail()
}

func (ui *recordsTUI) shiftColumns(delta int) {
	if delta == 0 {
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
	if ui.columnOffset+visibleColumnWindow >= len(ui.currentColumns()) {
		return
	}
	ui.columnOffset++
	ui.renderTable()
}

func (ui *recordsTUI) toggleDetail() {
	ui.detailVisible = !ui.detailVisible
	ui.applyDetailVisibility()
}

func (ui *recordsTUI) applyDetailVisibility() {
	if ui.detailVisible {
		ui.layout.ResizeItem(ui.detailView, 9, 0)
		return
	}
	ui.layout.ResizeItem(ui.detailView, 0, 0)
}

func (ui *recordsTUI) statusText() string {
	fields := "*"
	if len(ui.state.Fields) > 0 {
		fields = strings.Join(ui.state.Fields, ",")
	}

	text := fmt.Sprintf(
		"db=%s superuser=%s collection=%s page=%d perPage=%d totalItems=%d totalPages=%d filter=%q sort=%q fields=%s",
		ui.target.DB.Alias,
		ui.target.SU.Alias,
		ui.state.Collection,
		ui.state.Page,
		ui.state.PerPage,
		ui.totalItems,
		ui.totalPages,
		ui.state.Filter,
		ui.state.Sort,
		fields,
	)
	if strings.TrimSpace(ui.statusMessage) == "" {
		return text
	}
	return text + " status=" + ui.statusMessage
}

func (ui *recordsTUI) openInputModal(title, label, current string, apply func(string) error) {
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

	ui.showModal("input", 60, 9, form)
}

func (ui *recordsTUI) openColumnsModal() {
	cols := mergeColumns(ui.observedCols, nil)
	if len(cols) == 0 {
		ui.showError(apperr.Invalid("No columns available yet.", "Refresh results first."))
		return
	}

	selected := selectedColumns(cols, ui.state.Fields)
	form := tview.NewForm()
	for _, col := range cols {
		name := col
		form.AddCheckbox(name, selected[name], func(checked bool) {
			selected[name] = checked
		})
	}
	form.AddButton("Apply", func() {
		picked := pickedColumns(cols, selected)
		if len(picked) == 0 {
			ui.showError(apperr.Invalid("At least one column must be selected.", "Choose one or more columns."))
			return
		}
		ui.state.Fields = normalizeColumns(picked)
		ui.state.Page = 1
		ui.closeModal("columns")
		_ = ui.fetchAndRender()
	})
	form.AddButton("Clear", func() {
		ui.state.Fields = nil
		ui.state.Page = 1
		ui.closeModal("columns")
		_ = ui.fetchAndRender()
	})
	form.AddButton("Cancel", func() {
		ui.closeModal("columns")
	})
	form.SetBorder(true).SetTitle(" Columns ")

	height := len(cols) + 6
	if height > 24 {
		height = 24
	}
	ui.modalOpen = true
	ui.showModal("columns", 70, height, form)
}

func (ui *recordsTUI) openDBManagerModal() {
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
		record, ok := manager.chooseAlias(text)
		applyDBFormSelection(aliasField, baseURLField, record, ok)
	})
	record, ok := manager.chooseAlias(managerNewOption)
	applyDBFormSelection(aliasField, baseURLField, record, ok)

	form.AddButton("Save", func() {
		ui.closeModal("db-manager")
		updated, err := ui.saveDBManagerSelection(manager.selectedAlias, aliasField.GetText(), baseURLField.GetText())
		if err != nil {
			ui.showError(err)
			return
		}
		ui.syncDBManagerTarget(manager.selectedAlias, updated)
		ui.setStatusMessage("db aliases updated")
	})
	form.AddButton("Delete", func() {
		if manager.isCreate() {
			ui.showError(apperr.Invalid("Select an existing db alias first.", "Choose a saved db alias from the target field."))
			return
		}
		ui.closeModal("db-manager")
		if err := ui.dispatcher.removeDBAlias(manager.selectedAlias); err != nil {
			ui.showError(err)
			return
		}
		ui.setStatusMessage(dbDeleteStatus(ui.target.DB.Alias, manager.selectedAlias))
	})
	form.AddButton("Close", func() {
		ui.closeModal("db-manager")
	})
	form.SetBorder(true).SetTitle(" DB Aliases ")
	form.SetButtonsAlign(tview.AlignRight)

	ui.modalOpen = true
	ui.showModal("db-manager", 76, 12, form)
}

func (ui *recordsTUI) openSuperuserManagerModal() {
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
	if err := ui.reloadSuperuserManager(manager, manager.selectedDB); err != nil {
		ui.showError(err)
		return
	}

	form := tview.NewForm()
	form.AddDropDown("db", manager.dbChoices(), manager.selectedDBIndex(), nil)
	form.AddDropDown("target", manager.aliasChoices(), 0, nil)
	form.AddInputField("alias", "", 0, nil, nil)
	form.AddInputField("email", "", 0, nil, nil)
	form.AddPasswordField("password", "", 0, '*', nil)

	dbDropdown := form.GetFormItem(0).(*tview.DropDown)
	targetDropdown := form.GetFormItem(1).(*tview.DropDown)
	aliasField := form.GetFormItem(2).(*tview.InputField)
	emailField := form.GetFormItem(3).(*tview.InputField)
	passwordField := form.GetFormItem(4).(*tview.InputField)

	dbDropdown.SetSelectedFunc(func(text string, _ int) {
		if err := ui.reloadSuperuserManager(manager, text); err != nil {
			ui.showError(err)
			return
		}
		targetDropdown.SetOptions(manager.aliasChoices(), func(option string, _ int) {
			record, ok := manager.chooseAlias(option)
			applySuperuserFormSelection(aliasField, emailField, passwordField, record, ok)
		})
		targetDropdown.SetCurrentOption(0)
		record, ok := manager.chooseAlias(managerNewOption)
		applySuperuserFormSelection(aliasField, emailField, passwordField, record, ok)
	})
	targetDropdown.SetSelectedFunc(func(text string, _ int) {
		record, ok := manager.chooseAlias(text)
		applySuperuserFormSelection(aliasField, emailField, passwordField, record, ok)
	})
	record, ok := manager.chooseAlias(managerNewOption)
	applySuperuserFormSelection(aliasField, emailField, passwordField, record, ok)

	form.AddButton("Save", func() {
		ui.closeModal("superuser-manager")
		updated, err := ui.saveSuperuserManagerSelection(manager.selectedDB, manager.selectedAlias, aliasField.GetText(), emailField.GetText(), passwordField.GetText())
		if err != nil {
			ui.showError(err)
			return
		}
		ui.syncSuperuserManagerTarget(manager.selectedDB, manager.selectedAlias, updated)
		ui.setStatusMessage("superusers updated")
	})
	form.AddButton("Delete", func() {
		if manager.isCreate() {
			ui.showError(apperr.Invalid("Select an existing superuser first.", "Choose a saved superuser from the target field."))
			return
		}
		ui.closeModal("superuser-manager")
		if err := ui.dispatcher.removeSuperuser(manager.selectedDB, manager.selectedAlias); err != nil {
			ui.showError(err)
			return
		}
		ui.setStatusMessage(superuserDeleteStatus(ui.target, manager.selectedDB, manager.selectedAlias))
	})
	form.AddButton("Close", func() {
		ui.closeModal("superuser-manager")
	})
	form.SetBorder(true).SetTitle(" Superusers ")
	form.SetButtonsAlign(tview.AlignRight)

	ui.modalOpen = true
	ui.showModal("superuser-manager", 80, 14, form)
}

func (ui *recordsTUI) reloadSuperuserManager(manager *superuserManagerState, dbAlias string) error {
	items, err := ui.dispatcher.suStore.ListByDB(dbAlias)
	if err != nil {
		return mapStoreError(err)
	}
	manager.selectDB(dbAlias, items)
	return nil
}

func (ui *recordsTUI) saveDBManagerSelection(currentAlias, nextAlias, baseURL string) (storage.DB, error) {
	if strings.TrimSpace(currentAlias) == "" {
		return ui.dispatcher.saveDBAlias(nextAlias, baseURL)
	}
	return ui.dispatcher.updateDBAlias(currentAlias, nextAlias, baseURL)
}

func (ui *recordsTUI) saveSuperuserManagerSelection(dbAlias, currentAlias, nextAlias, email, password string) (storage.Superuser, error) {
	if strings.TrimSpace(currentAlias) == "" {
		return ui.dispatcher.saveSuperuser(dbAlias, nextAlias, email, password)
	}
	return ui.dispatcher.updateSuperuser(dbAlias, currentAlias, nextAlias, email, password)
}

func (ui *recordsTUI) syncDBManagerTarget(previousAlias string, updated storage.DB) {
	if strings.EqualFold(ui.target.DB.Alias, previousAlias) || (strings.TrimSpace(previousAlias) == "" && strings.EqualFold(ui.target.DB.Alias, updated.Alias)) {
		ui.target.DB = updated
	}
}

func (ui *recordsTUI) syncSuperuserManagerTarget(dbAlias, previousAlias string, updated storage.Superuser) {
	if !strings.EqualFold(ui.target.DB.Alias, dbAlias) {
		return
	}
	if strings.EqualFold(ui.target.SU.Alias, previousAlias) || (strings.TrimSpace(previousAlias) == "" && strings.EqualFold(ui.target.SU.Alias, updated.Alias)) {
		ui.target.SU = updated
	}
}

func (ui *recordsTUI) closeModal(name string) {
	ui.pages.RemovePage(name)
	ui.modalOpen = false
	ui.app.SetFocus(ui.tableView)
}

func (ui *recordsTUI) showModal(name string, width, height int, primitive tview.Primitive) {
	ui.pages.AddPage(name, center(width, height, primitive), true, true)
	ui.app.SetFocus(primitive)
}

func (ui *recordsTUI) showError(err error) {
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

	ui.showModal("error", 80, 12, container)
}

func (ui *recordsTUI) setStatusMessage(message string) {
	ui.statusMessage = message
	ui.statusView.SetText(ui.statusText())
}

func (ui *recordsTUI) observeColumns() {
	fresh := pocketbase.CollectColumns(ui.result.Rows)
	mergeColumns(ui.observedCols, fresh)
}

func (ui *recordsTUI) currentColumns() []string {
	if len(ui.state.Fields) > 0 {
		return ui.state.Fields
	}
	cols := pocketbase.CollectColumns(ui.result.Rows)
	if len(cols) == 0 {
		return []string{"result"}
	}
	return cols
}

func (ui *recordsTUI) visibleColumns() []string {
	all := ui.currentColumns()
	if len(all) == 0 {
		return []string{"result"}
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

func selectedColumns(cols, current []string) map[string]bool {
	selected := map[string]bool{}
	if len(current) == 0 {
		for _, col := range cols {
			selected[col] = true
		}
		return selected
	}
	for _, col := range current {
		selected[col] = true
	}
	return selected
}

func pickedColumns(cols []string, selected map[string]bool) []string {
	picked := make([]string, 0, len(selected))
	for _, col := range cols {
		if selected[col] {
			picked = append(picked, col)
		}
	}
	return picked
}

func newDBManagerState(items []storage.DB) *dbManagerState {
	return &dbManagerState{items: items}
}

func (m *dbManagerState) choices() []string {
	return append([]string{managerNewOption}, dbAliasOptions(m.items)...)
}

func (m *dbManagerState) chooseAlias(alias string) (storage.DB, bool) {
	m.selectedAlias = normalizeManagerSelection(alias)
	m.selectedRecord = storage.DB{}
	if m.selectedAlias == "" {
		return storage.DB{}, false
	}
	record, ok := findDB(m.items, m.selectedAlias)
	if ok {
		m.selectedRecord = record
	}
	return m.selectedRecord, ok
}

func (m *dbManagerState) isCreate() bool {
	return strings.TrimSpace(m.selectedAlias) == ""
}

func newSuperuserManagerState(dbs []storage.DB, currentDB string) *superuserManagerState {
	selectedDB := currentDB
	if indexDBAlias(dbs, currentDB) < 0 {
		selectedDB = dbs[0].Alias
	}
	return &superuserManagerState{dbs: dbs, selectedDB: selectedDB}
}

func (m *superuserManagerState) dbChoices() []string {
	return dbAliasOptions(m.dbs)
}

func (m *superuserManagerState) selectedDBIndex() int {
	index := indexDBAlias(m.dbs, m.selectedDB)
	if index < 0 {
		return 0
	}
	return index
}

func (m *superuserManagerState) selectDB(dbAlias string, superusers []storage.Superuser) {
	m.selectedDB = dbAlias
	m.superusers = superusers
	m.selectedAlias = ""
	m.selectedRecord = storage.Superuser{}
}

func (m *superuserManagerState) aliasChoices() []string {
	return append([]string{managerNewOption}, superuserAliasOptions(m.superusers)...)
}

func (m *superuserManagerState) chooseAlias(alias string) (storage.Superuser, bool) {
	m.selectedAlias = normalizeManagerSelection(alias)
	m.selectedRecord = storage.Superuser{}
	if m.selectedAlias == "" {
		return storage.Superuser{}, false
	}
	record, ok := findSuperuser(m.superusers, m.selectedAlias)
	if ok {
		m.selectedRecord = record
	}
	return m.selectedRecord, ok
}

func (m *superuserManagerState) isCreate() bool {
	return strings.TrimSpace(m.selectedAlias) == ""
}

func normalizeManagerSelection(alias string) string {
	if alias == managerNewOption {
		return ""
	}
	return strings.TrimSpace(alias)
}

func applyDBFormSelection(aliasField, baseURLField *tview.InputField, record storage.DB, ok bool) {
	if !ok {
		aliasField.SetText("")
		baseURLField.SetText("")
		return
	}
	aliasField.SetText(record.Alias)
	baseURLField.SetText(record.BaseURL)
}

func applySuperuserFormSelection(aliasField, emailField, passwordField *tview.InputField, record storage.Superuser, ok bool) {
	if !ok {
		aliasField.SetText("")
		emailField.SetText("")
		passwordField.SetText("")
		return
	}
	aliasField.SetText(record.Alias)
	emailField.SetText(record.Email)
	passwordField.SetText("")
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
