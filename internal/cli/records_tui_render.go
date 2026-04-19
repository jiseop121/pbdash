package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/jiseop121/pbdash/internal/pocketbase"
)

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


func (ui *navigatorTUI) focusMain() {
	if ui.isModalOpen() || ui.app == nil {
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

