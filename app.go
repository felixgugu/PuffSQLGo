package main

import (
	"context"

	"puffsql/internal/applog"
	"puffsql/internal/connection"
	"puffsql/internal/storage"
)

type App struct {
	ctx     context.Context
	store   *storage.Store
	manager *connection.Manager
	logger  *applog.Logger
}

func NewApp() (*App, error) {
	store, err := storage.New()
	if err != nil {
		return nil, err
	}
	return &App{
		store:   store,
		manager: connection.NewManager(store),
		logger:  applog.New(store.QueryLogPath(), store.AILogPath()),
	}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.logger.Init()
}

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
