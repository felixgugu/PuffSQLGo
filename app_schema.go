package main

import (
	"context"
	"time"

	"puffsql/internal/model"
)

func (a *App) GetDatabases(connectionID string) ([]model.DatabaseItem, error) {
	defer a.recoverPanic("schema.databases")
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		a.logError("schema.databases", err)
		return nil, err
	}
	items, err := server.GetDatabases(ctx)
	a.logError("schema.databases", err)
	return items, err
}

func (a *App) GetTables(
	connectionID string,
	database *string,
	schema *string,
) ([]model.TableItem, error) {
	defer a.recoverPanic("schema.tables")
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		a.logError("schema.tables", err)
		return nil, err
	}
	items, err := server.GetTables(ctx, database, schema)
	a.logError("schema.tables", err)
	return items, err
}

func (a *App) GetColumns(
	connectionID string,
	database *string,
	schema string,
	table string,
) ([]model.ColumnItem, error) {
	defer a.recoverPanic("schema.columns")
	ctx, cancel := context.WithTimeout(a.context(), 45*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		a.logError("schema.columns", err)
		return nil, err
	}
	items, err := server.GetColumns(ctx, database, schema, table)
	a.logError("schema.columns", err)
	return items, err
}

func (a *App) GetTableObjects(
	connectionID string,
	database *string,
	schema string,
	table string,
) (model.TableObjects, error) {
	defer a.recoverPanic("schema.table_objects")
	ctx, cancel := context.WithTimeout(a.context(), 20*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		a.logError("schema.table_objects", err)
		return model.TableObjects{}, err
	}
	objects, err := server.GetTableObjects(ctx, database, schema, table)
	a.logError("schema.table_objects", err)
	return objects, err
}

func (a *App) GetDatabaseSchema(
	connectionID string,
	database *string,
) ([]model.TableSchema, error) {
	defer a.recoverPanic("schema.database_schema")
	ctx, cancel := context.WithTimeout(a.context(), 90*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		a.logError("schema.database_schema", err)
		return nil, err
	}
	schema, err := server.GetDatabaseSchema(ctx, database)
	a.logError("schema.database_schema", err)
	return schema, err
}

func (a *App) GetForeignKeys(
	connectionID string,
	database *string,
	schema *string,
	table *string,
) ([]model.ForeignKeyItem, error) {
	defer a.recoverPanic("schema.foreign_keys")
	ctx, cancel := context.WithTimeout(a.context(), 60*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		a.logError("schema.foreign_keys", err)
		return nil, err
	}
	items, err := server.GetForeignKeys(ctx, database, schema, table)
	a.logError("schema.foreign_keys", err)
	return items, err
}

func (a *App) SwitchDatabase(connectionID string, database string) error {
	defer a.recoverPanic("schema.switch_database")
	ctx, cancel := context.WithTimeout(a.context(), 30*time.Second)
	defer cancel()
	server, err := a.manager.Server(ctx, connectionID)
	if err != nil {
		a.logError("schema.switch_database", err)
		return err
	}
	err = server.SwitchDatabase(ctx, database)
	a.logError("schema.switch_database", err)
	return err
}
