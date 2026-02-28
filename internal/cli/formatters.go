package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"multi-pocketbase-ui/internal/apperr"
	"multi-pocketbase-ui/internal/pocketbase"
)

func RenderQueryResult(format, outPath string, result pocketbase.QueryResult) (string, error) {
	normalized, err := pocketbase.ValidateFormat(format)
	if err != nil {
		return "", apperr.Invalid("Unsupported output format.", "Use one of: table, csv, markdown.")
	}
	if normalized == "table" && strings.TrimSpace(outPath) != "" {
		return "", apperr.Invalid("`--out` cannot be used when `--format` is `table`.", "Remove `--out` or switch to `csv`/`markdown`.")
	}
	if (normalized == "csv" || normalized == "markdown") && strings.TrimSpace(outPath) == "" {
		return "", apperr.Invalid("Missing required option `--out` when `--format` is `csv` or `markdown`.", "Example: --format csv --out ./records.csv")
	}

	columns := pocketbase.CollectColumns(result.Rows)
	if normalized == "table" {
		output := renderTable(columns, result.Rows)
		if result.Meta != nil {
			output = output + fmt.Sprintf("\npage=%d perPage=%d totalItems=%d totalPages=%d", result.Meta.Page, result.Meta.PerPage, result.Meta.TotalItems, result.Meta.TotalPages)
		}
		return output, nil
	}

	var content string
	switch normalized {
	case "csv":
		content, err = renderCSV(columns, result.Rows)
	case "markdown":
		content = renderMarkdown(columns, result.Rows)
	}
	if err != nil {
		return "", apperr.RuntimeErr("Could not render output.", "", err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return "", apperr.RuntimeErr(fmt.Sprintf("Could not write output file %q.", outPath), "Check directory existence and write permission.", err)
	}
	return fmt.Sprintf("Exported %d rows to %s (format=%s)", len(result.Rows), outPath, normalized), nil
}

func RenderTableRows(rows []map[string]any) string {
	cols := pocketbase.CollectColumns(rows)
	return renderTable(cols, rows)
}

func renderTable(columns []string, rows []map[string]any) string {
	if len(columns) == 0 {
		columns = []string{"result"}
	}

	widths := make([]int, len(columns))
	upper := make([]string, len(columns))
	for i, c := range columns {
		upper[i] = strings.ToUpper(strings.ReplaceAll(c, "_", " "))
		widths[i] = len(upper[i])
	}

	grid := make([][]string, len(rows))
	for i, row := range rows {
		grid[i] = make([]string, len(columns))
		for j, col := range columns {
			cell := formatValue(row[col])
			grid[i][j] = cell
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}

	var b strings.Builder
	separator := func() {
		b.WriteString("+")
		for _, w := range widths {
			b.WriteString(strings.Repeat("-", w+2))
			b.WriteString("+")
		}
		b.WriteString("\n")
	}
	rowWriter := func(vals []string) {
		b.WriteString("|")
		for i, v := range vals {
			b.WriteString(" ")
			b.WriteString(v)
			b.WriteString(strings.Repeat(" ", widths[i]-len(v)+1))
			b.WriteString("|")
		}
		b.WriteString("\n")
	}

	separator()
	rowWriter(upper)
	separator()
	for _, row := range grid {
		rowWriter(row)
	}
	separator()
	b.WriteString(fmt.Sprintf("%d rows", len(rows)))
	return b.String()
}

func renderCSV(columns []string, rows []map[string]any) (string, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.Write(columns); err != nil {
		return "", err
	}
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = formatValue(row[col])
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderMarkdown(columns []string, rows []map[string]any) string {
	if len(columns) == 0 {
		columns = []string{"result"}
	}
	var b strings.Builder
	b.WriteString("| ")
	b.WriteString(strings.Join(columns, " | "))
	b.WriteString(" |\n| ")
	for i := range columns {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString("---")
	}
	b.WriteString(" |\n")
	for _, row := range rows {
		cells := make([]string, len(columns))
		for i, col := range columns {
			cells[i] = strings.ReplaceAll(formatValue(row[col]), "|", "\\|")
		}
		b.WriteString("| ")
		b.WriteString(strings.Join(cells, " | "))
		b.WriteString(" |\n")
	}
	return b.String()
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case float64, float32, int, int64, int32, uint, uint64, bool:
		return fmt.Sprintf("%v", t)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}
