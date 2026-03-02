package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"multi-pocketbase-ui/internal/apperr"
	"multi-pocketbase-ui/internal/pocketbase"
)

const visibleColumnWindow = 8

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
}

func (d *Dispatcher) runRecordsTUI(ctx context.Context, target pbTarget, state RecordsQueryState) error {
	ui := &recordsTUI{
		dispatcher:    d,
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
	if err := ui.fetch(); err != nil {
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

func (ui *recordsTUI) setupViews() {
	ui.statusView.SetDynamicColors(true)
	ui.statusView.SetText(ui.statusText())

	ui.tableView.SetBorders(false)
	ui.tableView.SetSelectable(true, false)
	ui.tableView.SetFixed(1, 0)
	ui.tableView.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	ui.tableView.SetSelectionChangedFunc(func(row, _ int) {
		if row <= 0 {
			ui.selectedIndex = 0
			ui.renderDetail()
			return
		}
		ui.selectedIndex = row - 1
		ui.renderDetail()
	})

	ui.helpView.SetText("q/esc quit  j/k move  h/l or <-/-> horiz  / filter  s sort  c columns  [/] page  g/G first/last  r refresh  Enter detail")

	ui.detailView.SetTextAlign(tview.AlignLeft)
	ui.detailView.SetDynamicColors(true)
	ui.detailView.SetBorder(true)
	ui.detailView.SetTitle(" record detail ")

	ui.layout = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.layout.AddItem(ui.statusView, 1, 0, false)
	ui.layout.AddItem(ui.tableView, 0, 1, true)
	ui.layout.AddItem(ui.detailView, 9, 0, false)
	ui.layout.AddItem(ui.helpView, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", ui.layout, true, true)
	ui.pages.SetInputCapture(ui.handleKey)

	ui.renderTable()
	ui.renderDetail()
}

func (ui *recordsTUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.modalOpen {
		return event
	}

	switch event.Key() {
	case tcell.KeyEsc:
		ui.app.Stop()
		return nil
	case tcell.KeyEnter:
		ui.toggleDetail()
		return nil
	case tcell.KeyLeft:
		ui.shiftColumns(-1)
		return nil
	case tcell.KeyRight:
		ui.shiftColumns(1)
		return nil
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
		ui.shiftColumns(-1)
		return nil
	case 'l':
		ui.shiftColumns(1)
		return nil
	case '/':
		ui.openInputModal("Filter", "filter", ui.state.Filter, func(val string) error {
			ui.state.Filter = strings.TrimSpace(val)
			ui.state.Page = 1
			return ui.fetchAndRender()
		})
		return nil
	case 's':
		ui.openInputModal("Sort", "sort", ui.state.Sort, func(val string) error {
			ui.state.Sort = strings.TrimSpace(val)
			ui.state.Page = 1
			return ui.fetchAndRender()
		})
		return nil
	case 'c':
		ui.openColumnsModal()
		return nil
	case 'r':
		_ = ui.fetchAndRender()
		return nil
	case '[':
		if ui.state.Page > 1 {
			ui.state.Page--
			_ = ui.fetchAndRender()
		}
		return nil
	case ']':
		if ui.totalPages == 0 || ui.state.Page < ui.totalPages {
			ui.state.Page++
			_ = ui.fetchAndRender()
		}
		return nil
	case 'g':
		ui.state.Page = 1
		_ = ui.fetchAndRender()
		return nil
	case 'G':
		if ui.totalPages > 0 {
			ui.state.Page = ui.totalPages
			_ = ui.fetchAndRender()
		}
		return nil
	}
	return event
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
	all := ui.currentColumns()
	if ui.columnOffset+visibleColumnWindow >= len(all) {
		return
	}
	ui.columnOffset++
	ui.renderTable()
}

func (ui *recordsTUI) fetch() error {
	result, err := ui.dispatcher.fetchRecords(ui.ctx, ui.target, ui.state)
	if err != nil {
		return err
	}
	ui.result = result
	if result.Meta != nil {
		if result.Meta.Page > 0 {
			ui.state.Page = result.Meta.Page
		}
		if result.Meta.PerPage > 0 {
			ui.state.PerPage = result.Meta.PerPage
		}
		ui.totalPages = result.Meta.TotalPages
		ui.totalItems = result.Meta.TotalItems
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
	ui.statusView.SetText(ui.statusText())
	ui.renderTable()
	ui.renderDetail()
	return nil
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

func (ui *recordsTUI) toggleDetail() {
	ui.detailVisible = !ui.detailVisible
	if ui.detailVisible {
		ui.layout.ResizeItem(ui.detailView, 9, 0)
	} else {
		ui.layout.ResizeItem(ui.detailView, 0, 0)
	}
}

func (ui *recordsTUI) statusText() string {
	fields := "*"
	if len(ui.state.Fields) > 0 {
		fields = strings.Join(ui.state.Fields, ",")
	}
	return fmt.Sprintf("db=%s superuser=%s collection=%s page=%d perPage=%d totalItems=%d totalPages=%d filter=%q sort=%q fields=%s",
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

	modal := center(60, 9, form)
	ui.pages.AddPage("input", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *recordsTUI) openColumnsModal() {
	cols := mergeColumns(ui.observedCols, nil)
	if len(cols) == 0 {
		ui.showError(apperr.Invalid("No columns available yet.", "Refresh results first."))
		return
	}

	selected := map[string]bool{}
	if len(ui.state.Fields) == 0 {
		for _, col := range cols {
			selected[col] = true
		}
	} else {
		for _, col := range ui.state.Fields {
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

	h := len(cols) + 6
	if h > 24 {
		h = 24
	}
	modal := center(70, h, form)
	ui.pages.AddPage("columns", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *recordsTUI) closeModal(name string) {
	ui.pages.RemovePage(name)
	ui.modalOpen = false
	ui.app.SetFocus(ui.tableView)
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

	ui.pages.AddPage("error", center(80, 12, container), true, true)
	ui.app.SetFocus(form)
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
