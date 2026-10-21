package main

import (
	"context"
	"time"

	"puffsql/internal/model"
)

func (a *App) GetDatabases(connectionID string) ([]model.DatabaseItem, error) {
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return server.GetDatabases(ctx)
}

func (a *App) GetTables(
	connectionID string,
	database *string,
	schema *string,
) ([]model.TableItem, error) {
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return server.GetTables(ctx, database, schema)
}

func (a *App) GetColumns(
	connectionID string,
	database *string,
	schema string,
	table string,
) ([]model.ColumnItem, error) {
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return server.GetColumns(ctx, database, schema, table)
}

func (a *App) GetTableObjects(
	connectionID string,
	database *string,
	schema string,
	table string,
) (model.TableObjects, error) {
	ctx, cancel := context.WithTimeout(a.context(), 20*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return model.TableObjects{}, err
	}
	return server.GetTableObjects(ctx, database, schema, table)
}

func (a *App) GetDatabaseSchema(
	connectionID string,
	database *string,
) ([]model.TableSchema, error) {
	ctx, cancel := context.WithTimeout(a.context(), 90*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return server.GetDatabaseSchema(ctx, database)
}

func (a *App) GetForeignKeys(
	connectionID string,
	database *string,
	schema *string,
	table *string,
) ([]model.ForeignKeyItem, error) {
	ctx, cancel := context.WithTimeout(a.context(), 60*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return server.GetForeignKeys(ctx, database, schema, table)
}

func (a *App) SwitchDatabase(connectionID string, database string) error {
	ctx, cancel := context.WithTimeout(a.context(), 30*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		return err
	}
	return server.SwitchDatabase(ctx, database)
}
