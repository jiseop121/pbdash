package cli

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/jiseop121/pbdash/internal/apperr"
	"github.com/jiseop121/pbdash/internal/storage"
)

func (ui *navigatorTUI) openInputModal(title, label, current string, apply func(string) error) {
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
			ui.closeModal("columns")
			ui.showError(apperr.Invalid("At least one column must be selected.", "Choose one or more columns."))
			return
		}
		ui.recordsState.Fields = normalizeColumns(picked)
		ui.recordsState.Page = 1
		ui.closeModal("columns")
		if err := ui.fetchAndRenderRecords(); err != nil {
			ui.showError(err)
		}
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
			table.SetCell(0, 0, tview.NewTableCell("No entries yet. Press 'n' to add one."))
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
	var dbForm *tview.Form

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
		case tcell.KeyTab:
			ui.app.SetFocus(dbForm)
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

	dbForm = tview.NewForm()
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
		case tcell.KeyTab, tcell.KeyBacktab:
			ui.app.SetFocus(table)
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
	ui.focusMain()
}

func (ui *navigatorTUI) showError(err error) {
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
	ui.pages.AddPage(pageName, center(60, 7, container), true, true)
	ui.app.SetFocus(form)
}

func (ui *navigatorTUI) dismissErrorModal() {
	ui.pages.RemovePage("error")
	ui.focusMain()
}


func buildManagerTable(rows [][]string) *tview.Table {
	table := tview.NewTable().SetSelectable(true, false)
	table.SetBorderPadding(0, 0, 1, 1)
	if len(rows) == 0 {
		table.SetCell(0, 0, tview.NewTableCell("No entries yet. Press 'n' to add one."))
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
	if len(dbs) == 0 {
		return superuserManagerState{}
	}
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
