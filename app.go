package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"puffsql/internal/applog"
	"puffsql/internal/connection"
	"puffsql/internal/errlog"
	"puffsql/internal/storage"
)

type App struct {
	ctx       context.Context
	store     *storage.Store
	manager   *connection.Manager
	logger    *applog.Logger
	errLogger *errlog.Logger
}

func NewApp() (*App, error) {
	store, err := storage.New()
	if err != nil {
		return nil, err
	}
	errorLogPath, fallback := errlog.ResolvePath(store.Dir())
	errLogger := errlog.New(errorLogPath)
	if fallback {
		errLogger.Write(errlog.Entry{
			Level:   "WARNING",
			Source:  "app",
			Message: "executable directory is not writable; the error log is stored in the application data directory",
		})
	}
	return &App{
		store:     store,
		manager:   connection.NewManager(store, errLogger),
		logger:    applog.New(store.QueryLogPath(), store.AILogPath()),
		errLogger: errLogger,
	}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.logger.Init()
	executable, _ := os.Executable()
	a.errLogger.Write(errlog.Entry{
		Level:  "INFO",
		Source: "app",
		Message: fmt.Sprintf(
			"PuffSQL started; executable: %s; error log: %s; query log: %s",
			executable, a.errLogger.Path(), a.logger.QueryPath(),
		),
	})
}

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// logError records an error surfaced by a Wails command. It is a no-op when the logger could not
// be created, so logging can never break the command that produced the error.
func (a *App) logError(source string, err error) {
	if err == nil {
		return
	}
	a.errLogger.Write(errlog.Entry{
		Level:   "ERROR",
		Source:  source,
		Message: err.Error(),
	})
}

// recoverPanic records the stack of a panic before letting it continue to the Wails boundary.
func (a *App) recoverPanic(source string) {
	if recovered := recover(); recovered != nil {
		a.errLogger.Write(errlog.Entry{
			Level:   "PANIC",
			Source:  source,
			Message: fmt.Sprint(recovered),
			Stack:   string(debug.Stack()),
		})
		panic(recovered)
	}
}
