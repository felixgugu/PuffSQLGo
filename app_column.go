package main

import (
	"context"
	"time"

	"puffsql/internal/model"
)

func (a *App) PreviewColumnChange(req model.ColumnChangeRequest) (model.ColumnChangePreview, error) {
	server, err := a.manager.Server(a.context(), req.ConnectionID)
	if err != nil {
		return model.ColumnChangePreview{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	return server.PreviewColumnChange(ctx, req)
}

func (a *App) ApplyColumnChange(
	req model.ColumnChangeRequest,
	fingerprint string,
) (model.ColumnChangeResult, error) {
	server, err := a.manager.Server(a.context(), req.ConnectionID)
	if err != nil {
		return model.ColumnChangeResult{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), 2*time.Minute)
	defer cancel()
	return server.ApplyColumnChange(ctx, req, fingerprint)
}
