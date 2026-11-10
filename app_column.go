package main

import (
	"context"
	"time"

	"puffsql/internal/model"
)

func (a *App) PreviewColumnChange(req model.ColumnChangeRequest) (model.ColumnChangePreview, error) {
	defer a.recoverPanic("column.preview")
	server, err := a.manager.Server(a.context(), req.ConnectionID)
	if err != nil {
		a.logError("column.preview", err)
		return model.ColumnChangePreview{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	preview, err := server.PreviewColumnChange(ctx, req)
	a.logError("column.preview", err)
	return preview, err
}

func (a *App) ApplyColumnChange(
	req model.ColumnChangeRequest,
	fingerprint string,
) (model.ColumnChangeResult, error) {
	defer a.recoverPanic("column.apply")
	server, err := a.manager.Server(a.context(), req.ConnectionID)
	if err != nil {
		a.logError("column.apply", err)
		return model.ColumnChangeResult{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 2*time.Minute)
	defer cancel()
	result, err := server.ApplyColumnChange(ctx, req, fingerprint)
	a.logError("column.apply", err)
	return result, err
}
