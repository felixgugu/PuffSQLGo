package main

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"puffsql/internal/model"
)

const importProgressEvent = "sqlight:import-progress"

func (a *App) GetTableImportCapabilities(
	connectionID string,
	database *string,
	schema string,
	table string,
) (model.ImportCapabilities, error) {
	server, err := a.manager.Server(a.context(), connectionID)
	if err != nil {
		return model.ImportCapabilities{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 60*time.Second)
	defer cancel()
	return server.GetImportCapabilities(ctx, database, schema, table)
}

func (a *App) ImportTableRows(
	connectionID string,
	database *string,
	schema string,
	table string,
	columns []string,
	rows []model.ImportRowPayload,
	manualIdentity bool,
	importID *string,
) (model.ImportResult, error) {
	server, err := a.manager.Server(a.context(), connectionID)
	if err != nil {
		return model.ImportResult{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 10*time.Minute)
	defer cancel()
	id := ""
	if importID != nil {
		id = *importID
	}
	result := server.ImportRows(ctx, database, schema, table, columns, rows, manualIdentity,
		func(processed, total int) {
			runtime.EventsEmit(a.ctx, importProgressEvent, model.ImportProgressEvent{
				ImportID: id, ProcessedRows: processed, TotalRows: total,
			})
		})
	status := "SUCCESS"
	if result.RolledBack {
		status = "ERROR"
	}
	message := model.QueryMessage{
		Level: "info", Timestamp: time.Now().Format(time.RFC3339),
		Message: "TSV import completed",
	}
	if result.RolledBack {
		message.Level = "error"
		message.Message = "TSV import rolled back"
	}
	a.logger.LogQuery(
		connectionID, databaseValue(database),
		"-- TSV import into "+schema+"."+table,
		[]model.QueryMessage{message}, result.ExecutionTime, status,
	)
	return result, nil
}
