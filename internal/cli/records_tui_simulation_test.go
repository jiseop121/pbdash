package cli

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jiseop121/pbdash/internal/pocketbase"
	"github.com/jiseop121/pbdash/internal/storage"
)

const (
	simulationWidth  = 160
	simulationHeight = 30
)

type recordsSimulationHarness struct {
	t      *testing.T
	ui     *navigatorTUI
	screen tcell.SimulationScreen
}

type recordsSimulationConfig struct {
	result        pocketbase.QueryResult
	state         RecordsQueryState
	session        pbSession
	totalItems    int
	totalPages    int
	detailVisible bool
}

func TestNavigatorTUISimulationRendersRecordsScreen(t *testing.T) {
	h := newRecordsSimulationHarness(t, recordsSimulationConfig{
		result: pocketbase.QueryResult{Rows: []map[string]any{
			{"id": "rec-001", "title": "first-row", "status": "open"},
			{"id": "rec-002", "title": "second-row", "status": "closed"},
		}},
		state: RecordsQueryState{Collection: "posts", Page: 1, PerPage: 20},
		session: pbSession{
			DB: storage.DB{Alias: "dev", BaseURL: "http://127.0.0.1:8090"},
			SU: storage.Superuser{DBAlias: "dev", Alias: "root", Email: "root@example.com"},
		},
		totalItems:    2,
		totalPages:    1,
		detailVisible: false,
	})

	// 헬프바 확인: "h/l"은 ASCII라 공백 삽입 없이 그대로 렌더링됨
	h.waitForText("collection=posts", "page 1/1 (2 items)", "ID", "TITLE", "h/l")
	h.waitForMissing("record detail")
}

func TestNavigatorTUISimulationOpensSelectedRecordDetail(t *testing.T) {
	h := newRecordsSimulationHarness(t, recordsSimulationConfig{
		result: pocketbase.QueryResult{Rows: []map[string]any{
			{"id": "rec-001", "title": "first-row", "detail_token": "detail-first"},
			{"id": "rec-002", "title": "second-row", "detail_token": "detail-second"},
		}},
		state: RecordsQueryState{
			Collection: "posts",
			Page:       1,
			PerPage:    20,
			Fields:     []string{"id", "title"},
		},
		session:        pbSession{DB: storage.DB{Alias: "dev"}, SU: storage.Superuser{Alias: "root"}},
		totalItems:    2,
		totalPages:    1,
		detailVisible: false,
	})

	h.injectKey(tcell.KeyEnter)
	// "record detail" 보더 타이틀로 레코드 상세 화면 진입 확인
	h.waitForText(`"detail_token": "detail-first"`, "record detail")
	assert.Equal(t, screenRecordDetail, h.ui.screen)

	h.injectKey(tcell.KeyEsc)
	h.waitForText("ID", "TITLE")
	h.waitForMissing(`"detail_token": "detail-first"`)
	assert.Equal(t, screenRecords, h.ui.screen)
}

func TestNavigatorTUISimulationCopiesRecordDetail(t *testing.T) {
	h := newRecordsSimulationHarness(t, recordsSimulationConfig{
		result: pocketbase.QueryResult{Rows: []map[string]any{
			{"id": "rec-001", "title": "first-row", "detail_token": "detail-first"},
		}},
		state: RecordsQueryState{
			Collection: "posts",
			Page:       1,
			PerPage:    20,
			Fields:     []string{"id", "title"},
		},
		session:        pbSession{DB: storage.DB{Alias: "dev"}, SU: storage.Superuser{Alias: "root"}},
		totalItems:    1,
		totalPages:    1,
		detailVisible: false,
	})

	h.injectKey(tcell.KeyEnter)
	h.waitForText("record detail", `"detail_token": "detail-first"`)

	h.injectRune('y')
	assert.Contains(t, string(h.screen.GetClipboardData()), `"detail_token": "detail-first"`)
	h.waitForText("copied (OSC52)")
}

func TestNavigatorTUISimulationHorizontalScrollsColumns(t *testing.T) {
	h := newRecordsSimulationHarness(t, recordsSimulationConfig{
		result:        queryResultWithColumns(visibleColumnWindow + 1),
		state:         RecordsQueryState{Collection: "posts", Page: 1, PerPage: 20},
		session:        pbSession{DB: storage.DB{Alias: "dev"}, SU: storage.Superuser{Alias: "root"}},
		totalItems:    1,
		totalPages:    1,
		detailVisible: false,
	})

	h.waitForText("C00", "C07")
	h.waitForMissing("C08")

	h.injectRune('l')
	h.waitForText("C01", "C08")
	h.waitForMissing("C00")
}

func newRecordsSimulationHarness(t *testing.T, cfg recordsSimulationConfig) *recordsSimulationHarness {
	t.Helper()

	cfg = normalizeRecordsSimulationConfig(cfg)
	screen := newSimulationScreen(t)
	ui := newSimulationNavigatorTUI(cfg, screen)
	ui.setupViews()
	ui.app.SetRoot(ui.pages, true)
	ui.focusMain()
	ui.app.ForceDraw()

	h := &recordsSimulationHarness{t: t, ui: ui, screen: screen}
	initial := h.renderedText()
	require.NotEmpty(t, initial, "simulation screen did not render after ForceDraw\nstatus=%q\nhelp=%q", ui.statusText(), ui.helpText())

	t.Cleanup(func() {
		ui.app.Stop()
	})

	return h
}

func (h *recordsSimulationHarness) injectRune(r rune) {
	h.t.Helper()
	h.injectEvent(tcell.KeyRune, r)
}

func (h *recordsSimulationHarness) injectKey(key tcell.Key) {
	h.t.Helper()
	h.injectEvent(key, 0)
}

func (h *recordsSimulationHarness) injectEvent(key tcell.Key, r rune) {
	h.t.Helper()
	h.screen.InjectKey(key, r, tcell.ModNone)
	h.dispatchNextEvent()
}

func (h *recordsSimulationHarness) waitForText(parts ...string) {
	h.t.Helper()

	last := h.renderedText()
	for _, part := range parts {
		require.Contains(h.t, last, part, "rendered text did not contain expected part.\nexpected=%v\nscreen=\n%s", parts, last)
	}
}

func (h *recordsSimulationHarness) waitForMissing(parts ...string) {
	h.t.Helper()

	last := h.renderedText()
	for _, part := range parts {
		require.NotContains(h.t, last, part, "rendered text still contained forbidden part.\nforbidden=%v\nscreen=\n%s", parts, last)
	}
}

func (h *recordsSimulationHarness) dispatchNextEvent() {
	h.t.Helper()

	event := h.screen.PollEvent()
	require.NotNil(h.t, event)

	key, ok := event.(*tcell.EventKey)
	require.True(h.t, ok, "expected key event, got %T", event)

	next := h.ui.handleGlobalKey(key)
	if next != nil {
		next = h.ui.handleKey(next)
	}

	h.ui.app.ForceDraw()
}

func (h *recordsSimulationHarness) renderedText() string {
	h.t.Helper()

	cells, width, height := h.screen.GetContents()
	lines := make([]string, 0, height)
	for y := 0; y < height; y++ {
		var line strings.Builder
		for x := 0; x < width; x++ {
			cell := cells[y*width+x]
			r := ' '
			if len(cell.Runes) > 0 {
				r = cell.Runes[0]
			} else if len(cell.Bytes) > 0 {
				r = rune(cell.Bytes[0])
			}
			line.WriteRune(r)
		}
		lines = append(lines, strings.TrimRight(line.String(), " "))
	}

	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func normalizeRecordsSimulationConfig(cfg recordsSimulationConfig) recordsSimulationConfig {
	if cfg.state.Page == 0 {
		cfg.state.Page = 1
	}
	if cfg.state.PerPage == 0 {
		cfg.state.PerPage = len(cfg.result.Rows)
	}
	if cfg.totalItems == 0 {
		cfg.totalItems = len(cfg.result.Rows)
	}
	if cfg.totalPages == 0 {
		cfg.totalPages = 1
	}
	return cfg
}

func newSimulationScreen(t *testing.T) tcell.SimulationScreen {
	t.Helper()

	screen := tcell.NewSimulationScreen("UTF-8")
	require.NotNil(t, screen)
	return screen
}

func newSimulationNavigatorTUI(cfg recordsSimulationConfig, screen tcell.SimulationScreen) *navigatorTUI {
	app := tview.NewApplication()
	app.SetScreen(screen)
	screen.SetSize(simulationWidth, simulationHeight)

	return &navigatorTUI{
		app:           app,
		stop:          app.Stop,
		termScreen:    screen,
		statusView:    tview.NewTextView(),
		tableView:     tview.NewTable(),
		detailView:    tview.NewTextView(),
		helpView:      tview.NewTextView(),
		screen:        screenRecords,
		hasSession:     true,
		session:        cfg.session,
		recordsState:  cfg.state,
		result:        cfg.result,
		totalItems:    cfg.totalItems,
		totalPages:    cfg.totalPages,
		detailVisible: cfg.detailVisible,
		observedCols:  map[string]struct{}{},
	}
}
