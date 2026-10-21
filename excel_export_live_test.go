package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	"puffsql/internal/model"
)

func TestLiveExcelSchemaAndSelectedDataExport(t *testing.T) {
	if os.Getenv("PUFFSQL_LIVE") != "1" {
		t.Skip("set PUFFSQL_LIVE=1 to export the test database")
	}
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	app.Startup(ctx)
	profiles, err := app.GetConnections()
	if err != nil {
		t.Fatal(err)
	}
	profileID := os.Getenv("PUFFSQL_TEST_PROFILE")
	var profile model.ConnectionProfile
	for _, candidate := range profiles {
		if candidate.ID == profileID {
			profile = candidate
			break
		}
	}
	if profile.ID == "" || profile.Database != "TestDB1" {
		t.Fatal("TestDB1 profile is required")
	}
	if err := app.Connect(profile.ID); err != nil {
		t.Fatal(err)
	}
	defer app.Disconnect(profile.ID)
	server, err := app.manager.Server(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	testTable := "PuffSQL_ExcelSchemaOnly_" + time.Now().Format("150405000000000")
	secret := "XLSX_SECRET_SHOULD_NOT_APPEAR"
	if _, err := app.ExecuteQuery(
		profile.ID,
		&profile.Database,
		"CREATE TABLE [dbo].["+testTable+"] ([Secret] nvarchar(100) NOT NULL); "+
			"INSERT INTO [dbo].["+testTable+"] ([Secret]) VALUES (N'"+secret+"');",
		nil, nil, nil,
	); err != nil {
		t.Fatal(err)
	}
	defer app.ExecuteQuery(
		profile.ID,
		&profile.Database,
		"DROP TABLE IF EXISTS [dbo].["+testTable+"]",
		nil, nil, nil,
	)
	tables, err := app.GetTables(profile.ID, &profile.Database, nil)
	if err != nil {
		t.Fatal(err)
	}
	baseCount := 0
	for _, table := range tables {
		if strings.EqualFold(table.Kind, "BASE TABLE") {
			baseCount++
		}
	}
	path := filepath.Join(t.TempDir(), "database.xlsx")
	result, err := server.ExportDatabaseSchemaExcel(ctx, profile.Database, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.TableCount != baseCount || result.SheetCount != baseCount+1 || result.ColumnCount == 0 {
		t.Fatalf("result = %#v, base tables = %d", result, baseCount)
	}
	workbook, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	sheets := workbook.GetSheetList()
	if len(sheets) != baseCount+1 || sheets[0] != "Index" {
		t.Fatalf("sheets = %#v", sheets)
	}
	indexRows, err := workbook.GetRows("Index")
	if err != nil {
		t.Fatal(err)
	}
	if len(indexRows) != baseCount+1 {
		t.Fatalf("index rows = %d, want %d", len(indexRows), baseCount+1)
	}
	for _, sheet := range sheets[1:] {
		rows, err := workbook.GetRows(sheet)
		if err != nil {
			t.Fatalf("%s: %v", sheet, err)
		}
		if len(rows) == 0 {
			t.Fatalf("%s has no header row", sheet)
		}
		for _, row := range rows {
			for _, cell := range row {
				if cell == secret {
					t.Fatalf("schema workbook leaked row data from %s", sheet)
				}
			}
		}
	}

	selectedPath := filepath.Join(t.TempDir(), "selected.xlsx")
	selected := []model.ExcelTableRef{{Schema: "dbo", Name: testTable}}
	for _, table := range tables {
		if strings.EqualFold(table.Kind, "BASE TABLE") && len(selected) < 2 && table.Name != testTable {
			selected = append(selected, model.ExcelTableRef{Schema: table.Schema, Name: table.Name})
		}
	}
	dataResult, err := server.ExportDatabaseTablesExcel(ctx, profile.Database, selected, selectedPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if dataResult.TableCount != len(selected) || dataResult.SheetCount != len(selected)+1 {
		t.Fatalf("data export result = %#v", dataResult)
	}
	dataWorkbook, err := excelize.OpenFile(selectedPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dataWorkbook.Close()
	dataSheets := dataWorkbook.GetSheetList()
	if len(dataSheets) != len(selected)+1 || dataSheets[0] != "Index" {
		t.Fatalf("data sheets = %#v", dataSheets)
	}
	dataRows, err := dataWorkbook.GetRows(dataSheets[1])
	if err != nil {
		t.Fatal(err)
	}
	foundData := false
	for _, row := range dataRows {
		for _, cell := range row {
			if cell == secret {
				foundData = true
			}
		}
	}
	if !foundData {
		t.Fatal("selected table data was not exported")
	}
}
