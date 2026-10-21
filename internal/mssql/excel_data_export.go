package mssql

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"puffsql/internal/model"
)

const excelMaxDataRows = 1_048_575

// ExportDatabaseTablesExcel writes row data for explicitly selected base tables only.
func (s *Server) ExportDatabaseTablesExcel(
	ctx context.Context,
	database string,
	refs []model.ExcelTableRef,
	path string,
	progress func(model.ExcelExportProgress),
) (model.ExcelExportResult, error) {
	started := time.Now()
	result := model.ExcelExportResult{FilePath: path}
	allTables, err := s.GetTables(ctx, &database, nil)
	if err != nil {
		return result, err
	}
	available := map[string]model.TableItem{}
	for _, table := range allTables {
		if strings.EqualFold(table.Kind, "BASE TABLE") {
			available[strings.ToLower(table.Schema+"\x00"+table.Name)] = table
		}
	}
	selected := make([]model.TableItem, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		key := strings.ToLower(strings.TrimSpace(ref.Schema) + "\x00" + strings.TrimSpace(ref.Name))
		table, ok := available[key]
		if !ok {
			return result, fmt.Errorf("table %s.%s is not available or is a view", ref.Schema, ref.Name)
		}
		if !seen[key] {
			seen[key] = true
			selected = append(selected, table)
		}
	}
	if len(selected) == 0 {
		return result, fmt.Errorf("select at least one table")
	}

	result.TableCount = len(selected)
	result.SheetCount = len(selected) + 1
	workbook := excelize.NewFile()
	defer workbook.Close()
	const indexSheet = "Index"
	if err := workbook.SetSheetName("Sheet1", indexSheet); err != nil {
		return result, err
	}
	styles, err := createExcelStyles(workbook)
	if err != nil {
		return result, err
	}
	usedNames := map[string]bool{strings.ToLower(indexSheet): true}
	indexInfo := make([]excelSheetInfo, 0, len(selected))
	for i, table := range selected {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		sheet := uniqueExcelSheetName(table.Schema+"."+table.Name, usedNames)
		info, err := s.writeExcelTableData(ctx, workbook, database, table, sheet, styles)
		if err != nil {
			return result, fmt.Errorf("export %s.%s: %w", table.Schema, table.Name, err)
		}
		indexInfo = append(indexInfo, info)
		result.ColumnCount += info.Columns
		result.TotalRows += info.Rows
		if info.Truncated {
			result.TruncatedCount++
		}
		if progress != nil {
			progress(model.ExcelExportProgress{
				Current: i + 1, Total: len(selected), Table: table.Schema + "." + table.Name,
				Sheets: len(indexInfo), Rows: result.TotalRows, Columns: result.ColumnCount,
			})
		}
	}
	if err := writeExcelDataIndex(workbook, indexSheet, styles, indexInfo); err != nil {
		return result, err
	}
	indexID, _ := workbook.GetSheetIndex(indexSheet)
	workbook.SetActiveSheet(indexID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return result, err
	}
	if err := workbook.SaveAs(path); err != nil {
		return result, err
	}
	result.ExecutionTimeMS = elapsedMillis(started)
	return result, nil
}

func (s *Server) writeExcelTableData(
	ctx context.Context,
	workbook *excelize.File,
	database string,
	table model.TableItem,
	sheet string,
	styles excelStyles,
) (excelSheetInfo, error) {
	info := excelSheetInfo{Schema: table.Schema, Table: table.Name, Sheet: sheet}
	if _, err := workbook.NewSheet(sheet); err != nil {
		return info, err
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return info, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "USE "+QuoteIdentifier(database)); err != nil {
		return info, err
	}
	rows, err := conn.QueryContext(ctx, "SELECT * FROM "+QualifiedTable(table.Schema, table.Name))
	if err != nil {
		return info, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return info, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return info, err
	}
	info.Columns = len(columns)
	lastColumn, err := excelize.ColumnNumberToName(max(len(columns), 1))
	if err != nil {
		return info, err
	}
	writer, err := workbook.NewStreamWriter(sheet)
	if err != nil {
		return info, err
	}
	header := make([]any, len(columns))
	for i, column := range columns {
		header[i] = column
	}
	if err := writer.SetRow("A1", header); err != nil {
		return info, err
	}
	values := make([]any, len(columns))
	scans := make([]any, len(columns))
	for i := range values {
		scans[i] = &values[i]
	}
	rowNumber := 2
	for rows.Next() {
		if info.Rows >= excelMaxDataRows {
			info.Truncated = true
			break
		}
		if err := rows.Scan(scans...); err != nil {
			return info, err
		}
		row := make([]any, len(values))
		for i, value := range values {
			typeName := ""
			if i < len(types) {
				typeName = types[i].DatabaseTypeName()
			}
			row[i] = excelCellValue(normalizeValue(value, typeName))
		}
		if err := writer.SetRow(fmt.Sprintf("A%d", rowNumber), row); err != nil {
			return info, err
		}
		rowNumber++
		info.Rows++
	}
	if err := rows.Err(); err != nil {
		return info, err
	}
	if err := writer.Flush(); err != nil {
		return info, err
	}
	_ = workbook.SetCellStyle(sheet, "A1", lastColumn+"1", styles.Header)
	_ = workbook.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	if len(columns) > 0 {
		_ = workbook.AutoFilter(sheet, "A1:"+lastColumn+"1", nil)
	}
	_ = workbook.SetColWidth(sheet, "A", lastColumn, 18)
	return info, nil
}

func writeExcelDataIndex(
	workbook *excelize.File,
	sheet string,
	styles excelStyles,
	items []excelSheetInfo,
) error {
	headers := []any{"#", "Schema", "Table", "Sheet", "Columns", "Rows", "Truncated"}
	if err := workbook.SetSheetRow(sheet, "A1", &headers); err != nil {
		return err
	}
	for i, item := range items {
		row := i + 2
		values := []any{i + 1, item.Schema, item.Table, item.Sheet, item.Columns, item.Rows, boolText(item.Truncated)}
		if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &values); err != nil {
			return err
		}
		_ = workbook.SetCellHyperLink(sheet, fmt.Sprintf("D%d", row), "'"+strings.ReplaceAll(item.Sheet, "'", "''")+"'!A1", "Location")
	}
	lastRow := max(len(items)+1, 1)
	_ = workbook.SetCellStyle(sheet, "A1", fmt.Sprintf("G%d", lastRow), styles.Wrap)
	_ = workbook.SetCellStyle(sheet, "A1", "G1", styles.Header)
	_ = workbook.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	_ = workbook.AutoFilter(sheet, fmt.Sprintf("A1:G%d", lastRow), nil)
	_ = workbook.SetColWidth(sheet, "A", "A", 6)
	_ = workbook.SetColWidth(sheet, "B", "D", 24)
	_ = workbook.SetColWidth(sheet, "E", "G", 14)
	_ = workbook.SetRowHeight(sheet, 1, 26)
	return nil
}

func excelCellValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return ""
	case model.BinaryCell:
		return fmt.Sprintf("[Binary %d bytes]", typed.Length)
	case int64:
		if typed > 999_999_999_999_999 || typed < -999_999_999_999_999 {
			return fmt.Sprint(typed)
		}
		return typed
	case uint64:
		if typed > math.MaxInt64 {
			return fmt.Sprint(typed)
		}
		return excelCellValue(int64(typed))
	default:
		return typed
	}
}
