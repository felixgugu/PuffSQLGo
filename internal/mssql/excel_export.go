package mssql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
	"puffsql/internal/model"
)

type excelSheetInfo struct {
	Schema      string
	Table       string
	Sheet       string
	Columns     int
	PrimaryKeys int
	Indexes     int
	Triggers    int
	Constraints int
	Rows        int64
	Truncated   bool
}

type excelStyles struct {
	Header  int
	Section int
	Wrap    int
}

// ExportDatabaseSchemaExcel writes one schema-only worksheet per base table.
// SELECT queries are never issued against the user tables; GetTableObjects reads catalog views only.
func (s *Server) ExportDatabaseSchemaExcel(
	ctx context.Context,
	database string,
	path string,
	progress func(model.ExcelExportProgress),
) (model.ExcelExportResult, error) {
	started := time.Now()
	result := model.ExcelExportResult{FilePath: path}
	tables, err := s.GetTables(ctx, &database, nil)
	if err != nil {
		return result, err
	}
	baseTables := make([]model.TableItem, 0, len(tables))
	for _, table := range tables {
		if strings.EqualFold(table.Kind, "BASE TABLE") {
			baseTables = append(baseTables, table)
		}
	}
	result.TableCount = len(baseTables)
	result.SheetCount = len(baseTables) + 1

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
	indexInfo := make([]excelSheetInfo, 0, len(baseTables))
	usedNames := map[string]bool{strings.ToLower(indexSheet): true}

	for i, table := range baseTables {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		sheet := uniqueExcelSheetName(table.Schema+"."+table.Name, usedNames)
		info, err := s.writeTableSchemaSheet(ctx, workbook, database, table, sheet, styles)
		if err != nil {
			return result, fmt.Errorf("export schema for %s.%s: %w", table.Schema, table.Name, err)
		}
		indexInfo = append(indexInfo, info)
		result.ColumnCount += info.Columns
		if progress != nil {
			progress(model.ExcelExportProgress{
				Current: i + 1,
				Total:   len(baseTables),
				Table:   table.Schema + "." + table.Name,
				Sheets:  len(indexInfo),
				Columns: result.ColumnCount,
			})
		}
	}

	if err := writeExcelIndex(workbook, indexSheet, styles, indexInfo); err != nil {
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

func (s *Server) writeTableSchemaSheet(
	ctx context.Context,
	workbook *excelize.File,
	database string,
	table model.TableItem,
	sheet string,
	styles excelStyles,
) (excelSheetInfo, error) {
	objects, err := s.GetTableObjects(ctx, &database, table.Schema, table.Name)
	if err != nil {
		return excelSheetInfo{}, err
	}
	if _, err := workbook.NewSheet(sheet); err != nil {
		return excelSheetInfo{}, err
	}
	info := excelSheetInfo{
		Schema: table.Schema, Table: table.Name, Sheet: sheet,
		Columns: len(objects.Columns), Indexes: len(objects.Indexes),
		Triggers: len(objects.Triggers), Constraints: len(objects.Constraints),
	}
	for _, column := range objects.Columns {
		if column.IsPrimaryKey {
			info.PrimaryKeys++
		}
	}

	headers := []any{
		"#", "Column Name", "Display Name / Description", "Data Type",
		"Length", "Precision", "Scale", "Nullable", "PK", "Identity",
		"Computed", "Default", "Collation",
	}
	if err := workbook.SetSheetRow(sheet, "A1", &headers); err != nil {
		return info, err
	}
	row := 2
	for i, column := range objects.Columns {
		values := []any{
			i + 1,
			column.Name,
			pointerText(column.Description),
			column.DataType,
			nullableIntValue(column.MaxLength),
			nullableIntValue(column.Precision),
			nullableIntValue(column.Scale),
			boolText(column.IsNullable),
			boolText(column.IsPrimaryKey),
			boolText(column.IsIdentity),
			boolText(column.IsComputed),
			pointerText(column.DefaultDefinition),
			pointerText(column.Collation),
		}
		if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &values); err != nil {
			return info, err
		}
		row++
	}
	lastColumnRow := max(row-1, 1)
	_ = workbook.SetCellStyle(sheet, "A1", fmt.Sprintf("M%d", lastColumnRow), styles.Wrap)
	_ = workbook.SetCellStyle(sheet, "A1", "M1", styles.Header)
	_ = workbook.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	_ = workbook.AutoFilter(sheet, fmt.Sprintf("A1:M%d", lastColumnRow), nil)

	row += 2
	row = writeSectionHeader(workbook, sheet, row, "Constraints", styles.Section)
	constraintHeaders := []any{
		"Kind", "Name", "Key Columns / Column", "Definition / Reference",
		"Clustered", "Disabled", "Not Trusted", "On Delete", "On Update",
	}
	if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &constraintHeaders); err != nil {
		return info, err
	}
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("I%d", row), styles.Header)
	row++
	for _, constraint := range objects.Constraints {
		keyColumns := make([]string, 0, len(constraint.KeyColumns))
		for _, key := range constraint.KeyColumns {
			keyColumns = append(keyColumns, key.Name)
		}
		reference := pointerText(constraint.Definition)
		if constraint.RefTable != nil {
			ref := pointerText(constraint.RefSchema) + "." + pointerText(constraint.RefTable)
			if len(constraint.RefColumns) > 0 {
				ref += " (" + strings.Join(constraint.RefColumns, ", ") + ")"
			}
			reference = ref
		}
		values := []any{
			constraint.Kind, constraint.Name, strings.Join(keyColumns, ", "), reference,
			optionalBoolText(constraint.IsClustered), boolText(constraint.IsDisabled),
			boolText(constraint.IsNotTrusted), pointerText(constraint.OnDelete),
			pointerText(constraint.OnUpdate),
		}
		if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &values); err != nil {
			return info, err
		}
		row++
	}

	row++
	row = writeSectionHeader(workbook, sheet, row, "Indexes", styles.Section)
	indexHeaders := []any{"Name", "Type", "Unique", "Primary Key", "Unique Constraint", "Disabled", "Columns", "Filter"}
	if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &indexHeaders); err != nil {
		return info, err
	}
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), styles.Header)
	row++
	for _, index := range objects.Indexes {
		columns := make([]string, 0, len(index.Columns))
		for _, column := range index.Columns {
			name := column.Name
			if column.Descending {
				name += " DESC"
			}
			if column.IsIncluded {
				name += " (INCLUDED)"
			}
			columns = append(columns, name)
		}
		values := []any{
			index.Name, index.TypeDesc, boolText(index.IsUnique), boolText(index.IsPrimaryKey),
			boolText(index.IsUniqueConstraint), boolText(index.IsDisabled),
			strings.Join(columns, ", "), pointerText(index.FilterDefinition),
		}
		if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &values); err != nil {
			return info, err
		}
		row++
	}

	row++
	row = writeSectionHeader(workbook, sheet, row, "Triggers", styles.Section)
	triggerHeaders := []any{"Name", "Events", "Instead Of", "Disabled", "Definition"}
	if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &triggerHeaders); err != nil {
		return info, err
	}
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styles.Header)
	row++
	for _, trigger := range objects.Triggers {
		values := []any{
			trigger.Name, strings.Join(trigger.Events, ", "), boolText(trigger.IsInsteadOf),
			boolText(trigger.IsDisabled), pointerText(trigger.Definition),
		}
		if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &values); err != nil {
			return info, err
		}
		row++
	}
	_ = workbook.SetColWidth(sheet, "A", "A", 6)
	_ = workbook.SetColWidth(sheet, "B", "B", 24)
	_ = workbook.SetColWidth(sheet, "C", "C", 28)
	_ = workbook.SetColWidth(sheet, "D", "I", 16)
	_ = workbook.SetColWidth(sheet, "J", "M", 22)
	_ = workbook.SetColWidth(sheet, "N", "N", 40)
	_ = workbook.SetRowHeight(sheet, 1, 26)
	return info, nil
}

func writeExcelIndex(
	workbook *excelize.File,
	sheet string,
	styles excelStyles,
	items []excelSheetInfo,
) error {
	headers := []any{"#", "Schema", "Table", "Sheet", "Columns", "PK Columns", "Constraints", "Indexes", "Triggers"}
	if err := workbook.SetSheetRow(sheet, "A1", &headers); err != nil {
		return err
	}
	for i, item := range items {
		row := i + 2
		values := []any{
			i + 1, item.Schema, item.Table, item.Sheet, item.Columns,
			item.PrimaryKeys, item.Constraints, item.Indexes, item.Triggers,
		}
		if err := workbook.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &values); err != nil {
			return err
		}
		_ = workbook.SetCellHyperLink(
			sheet,
			fmt.Sprintf("D%d", row),
			"'"+strings.ReplaceAll(item.Sheet, "'", "''")+"'!A1",
			"Location",
		)
	}
	lastRow := max(len(items)+1, 1)
	_ = workbook.SetCellStyle(sheet, "A1", fmt.Sprintf("I%d", lastRow), styles.Wrap)
	_ = workbook.SetCellStyle(sheet, "A1", "I1", styles.Header)
	_ = workbook.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	_ = workbook.AutoFilter(sheet, fmt.Sprintf("A1:I%d", lastRow), nil)
	_ = workbook.SetColWidth(sheet, "A", "A", 6)
	_ = workbook.SetColWidth(sheet, "B", "C", 24)
	_ = workbook.SetColWidth(sheet, "D", "D", 34)
	_ = workbook.SetColWidth(sheet, "E", "I", 14)
	_ = workbook.SetRowHeight(sheet, 1, 26)
	return nil
}

func createExcelStyles(workbook *excelize.File) (excelStyles, error) {
	header, err := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"235A7E"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	if err != nil {
		return excelStyles{}, err
	}
	section, err := workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "1E4F70", Size: 13},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"DCEAF4"}},
	})
	if err != nil {
		return excelStyles{}, err
	}
	wrap, err := workbook.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		return excelStyles{}, err
	}
	return excelStyles{Header: header, Section: section, Wrap: wrap}, nil
}

func writeSectionHeader(workbook *excelize.File, sheet string, row int, title string, style int) int {
	cell := fmt.Sprintf("A%d", row)
	_ = workbook.SetCellValue(sheet, cell, title)
	_ = workbook.MergeCell(sheet, cell, fmt.Sprintf("M%d", row))
	_ = workbook.SetCellStyle(sheet, cell, fmt.Sprintf("M%d", row), style)
	return row + 1
}

func uniqueExcelSheetName(candidate string, used map[string]bool) string {
	cleaned := strings.Trim(candidate, "'")
	replacer := strings.NewReplacer("[", "_", "]", "_", ":", "_", "*", "_", "?", "_", "/", "_", "\\", "_")
	cleaned = strings.TrimSpace(replacer.Replace(cleaned))
	if cleaned == "" {
		cleaned = "Table"
	}
	cleaned = truncateRunes(cleaned, 31)
	name := cleaned
	for suffix := 2; used[strings.ToLower(name)]; suffix++ {
		tail := fmt.Sprintf("_%d", suffix)
		name = truncateRunes(cleaned, 31-len(tail)) + tail
	}
	used[strings.ToLower(name)] = true
	return name
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func pointerText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullableIntValue(value *int) any {
	if value == nil {
		return ""
	}
	return *value
}

func boolText(value bool) string {
	if value {
		return "YES"
	}
	return "NO"
}

func optionalBoolText(value *bool) string {
	if value == nil {
		return ""
	}
	return boolText(*value)
}
