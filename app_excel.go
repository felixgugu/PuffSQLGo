package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"puffsql/internal/model"
)

const excelExportProgressEvent = "puffsql:excel-export-progress"

func (a *App) ExportDatabaseTablesToExcel(
	connectionID string,
	database string,
	tables []model.ExcelTableRef,
) (model.ExcelExportResult, error) {
	path, canceled, err := a.pickExcelPath(database, "匯出選取的 Table 資料至 Excel", "tables")
	if err != nil || canceled {
		return model.ExcelExportResult{Canceled: canceled}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 30*time.Minute)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return model.ExcelExportResult{}, err
	}
	return server.ExportDatabaseTablesExcel(ctx, database, tables, path, a.emitExcelProgress)
}

func (a *App) ExportDatabaseSchemaToExcel(
	connectionID string,
	database string,
) (model.ExcelExportResult, error) {
	path, canceled, err := a.pickExcelPath(database, "匯出資料庫完整 Table Schema 至 Excel", "schema")
	if err != nil || canceled {
		return model.ExcelExportResult{Canceled: canceled}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 10*time.Minute)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return model.ExcelExportResult{}, err
	}
	return server.ExportDatabaseSchemaExcel(ctx, database, path, a.emitExcelProgress)
}

func (a *App) pickExcelPath(database, title, suffix string) (string, bool, error) {
	if a.ctx == nil {
		return "", false, fmt.Errorf("application is not ready")
	}
	database = strings.TrimSpace(database)
	if database == "" {
		return "", false, fmt.Errorf("database is required")
	}
	safeName := strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	).Replace(database)
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: fmt.Sprintf("%s_%s_%s.xlsx", safeName, suffix, time.Now().Format("20060102_150405")),
		Filters: []runtime.FileFilter{{
			DisplayName: "Excel Workbook (*.xlsx)",
			Pattern:     "*.xlsx",
		}},
	})
	if err != nil {
		return "", false, err
	}
	if path == "" {
		return "", true, nil
	}
	if !strings.EqualFold(filepath.Ext(path), ".xlsx") {
		path += ".xlsx"
	}
	return path, false, nil
}

func (a *App) emitExcelProgress(progress model.ExcelExportProgress) {
	runtime.EventsEmit(a.ctx, excelExportProgressEvent, progress)
}
