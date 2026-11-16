package mysql

import (
	"context"
	"database/sql"
	"strings"

	"puffsql/internal/model"
)

// GetTableObjects reads the columns, constraints, indexes and triggers of one table from
// information_schema. SQL Server-only attributes (clustered, not trusted, filtered, included
// columns, instead-of triggers) keep their zero values.
func (s *Server) GetTableObjects(
	ctx context.Context,
	database *string,
	schema, table string,
) (model.TableObjects, error) {
	target := s.tableDatabase(database, schema)
	objects := model.TableObjects{
		Columns:     []model.ColumnItem{},
		Constraints: []model.TableConstraintItem{},
		Indexes:     []model.TableIndexItem{},
		Triggers:    []model.TableTriggerItem{},
	}
	columns, err := s.GetColumns(ctx, &target, target, table)
	if err != nil {
		return objects, err
	}
	objects.Columns = columns

	if objects.Indexes, err = s.tableIndexes(ctx, target, table); err != nil {
		return objects, err
	}
	// In MySQL a primary key or unique constraint is its index.
	for _, index := range objects.Indexes {
		if !index.IsUnique {
			continue
		}
		kind := "UNIQUE"
		if index.IsPrimaryKey {
			kind = "PRIMARY_KEY"
		}
		keys := make([]model.KeyColumnRef, 0, len(index.Columns))
		for _, column := range index.Columns {
			keys = append(keys, model.KeyColumnRef{Name: column.Name, Descending: column.Descending})
		}
		objects.Constraints = append(objects.Constraints, model.TableConstraintItem{
			Name: index.Name, Kind: kind, KeyColumns: keys, RefColumns: []string{},
		})
	}

	foreignKeys, err := s.tableForeignKeys(ctx, target, table)
	if err != nil {
		return objects, err
	}
	objects.Constraints = append(objects.Constraints, foreignKeys...)
	// CHECK_CONSTRAINTS only exists on MySQL 8.0.16+ / MariaDB 10.2+; older servers simply
	// have no check constraints to report.
	if checks, err := s.tableChecks(ctx, target, table); err == nil {
		objects.Constraints = append(objects.Constraints, checks...)
	}

	if objects.Triggers, err = s.tableTriggers(ctx, target, table); err != nil {
		return objects, err
	}
	return objects, nil
}

func (s *Server) tableIndexes(ctx context.Context, database, table string) ([]model.TableIndexItem, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT INDEX_NAME, NON_UNIQUE, INDEX_TYPE, COLUMN_NAME, COLLATION
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
ORDER BY INDEX_NAME, SEQ_IN_INDEX`, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.TableIndexItem{}
	position := map[string]int{}
	for rows.Next() {
		var name, indexType string
		var nonUnique int
		var column, collation sql.NullString
		if err := rows.Scan(&name, &nonUnique, &indexType, &column, &collation); err != nil {
			return nil, err
		}
		at, exists := position[name]
		if !exists {
			at = len(items)
			position[name] = at
			primary := strings.EqualFold(name, "PRIMARY")
			items = append(items, model.TableIndexItem{
				Name: name, TypeDesc: indexType,
				IsUnique: nonUnique == 0, IsPrimaryKey: primary,
				IsUniqueConstraint: nonUnique == 0 && !primary,
				Columns:            []model.IndexColumnRef{},
			})
		}
		columnName := column.String
		if !column.Valid {
			// Functional key parts have no column name.
			columnName = "(expression)"
		}
		items[at].Columns = append(items[at].Columns, model.IndexColumnRef{
			Name: columnName, Descending: strings.EqualFold(collation.String, "D"),
		})
	}
	return items, rows.Err()
}

func (s *Server) tableForeignKeys(ctx context.Context, database, table string) ([]model.TableConstraintItem, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT k.CONSTRAINT_NAME, k.COLUMN_NAME, k.REFERENCED_TABLE_SCHEMA, k.REFERENCED_TABLE_NAME,
       k.REFERENCED_COLUMN_NAME, r.DELETE_RULE, r.UPDATE_RULE
FROM information_schema.KEY_COLUMN_USAGE k
LEFT JOIN information_schema.REFERENTIAL_CONSTRAINTS r
  ON r.CONSTRAINT_SCHEMA = k.CONSTRAINT_SCHEMA AND r.CONSTRAINT_NAME = k.CONSTRAINT_NAME
 AND r.TABLE_NAME = k.TABLE_NAME
WHERE k.TABLE_SCHEMA = ? AND k.TABLE_NAME = ? AND k.REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY k.CONSTRAINT_NAME, k.ORDINAL_POSITION`, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.TableConstraintItem{}
	position := map[string]int{}
	for rows.Next() {
		var name, column string
		var refSchema, refTable, refColumn, onDelete, onUpdate sql.NullString
		if err := rows.Scan(&name, &column, &refSchema, &refTable, &refColumn, &onDelete, &onUpdate); err != nil {
			return nil, err
		}
		at, exists := position[name]
		if !exists {
			at = len(items)
			position[name] = at
			items = append(items, model.TableConstraintItem{
				Name: name, Kind: "FOREIGN_KEY",
				KeyColumns: []model.KeyColumnRef{}, RefColumns: []string{},
				RefSchema: optionalText(refSchema.String), RefTable: optionalText(refTable.String),
				OnDelete: referentialAction(onDelete.String), OnUpdate: referentialAction(onUpdate.String),
			})
		}
		items[at].KeyColumns = append(items[at].KeyColumns, model.KeyColumnRef{Name: column})
		if refColumn.Valid {
			items[at].RefColumns = append(items[at].RefColumns, refColumn.String)
		}
	}
	return items, rows.Err()
}

func (s *Server) tableChecks(ctx context.Context, database, table string) ([]model.TableConstraintItem, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT tc.CONSTRAINT_NAME, cc.CHECK_CLAUSE
FROM information_schema.TABLE_CONSTRAINTS tc
JOIN information_schema.CHECK_CONSTRAINTS cc
  ON cc.CONSTRAINT_SCHEMA = tc.CONSTRAINT_SCHEMA AND cc.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
WHERE tc.TABLE_SCHEMA = ? AND tc.TABLE_NAME = ? AND tc.CONSTRAINT_TYPE = 'CHECK'
ORDER BY tc.CONSTRAINT_NAME`, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.TableConstraintItem{}
	for rows.Next() {
		var name string
		var clause sql.NullString
		if err := rows.Scan(&name, &clause); err != nil {
			return nil, err
		}
		items = append(items, model.TableConstraintItem{
			Name: name, Kind: "CHECK", Definition: optionalText(clause.String),
			KeyColumns: []model.KeyColumnRef{}, RefColumns: []string{},
		})
	}
	return items, rows.Err()
}

func (s *Server) tableTriggers(ctx context.Context, database, table string) ([]model.TableTriggerItem, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT TRIGGER_NAME, ACTION_TIMING, EVENT_MANIPULATION, ACTION_STATEMENT
FROM information_schema.TRIGGERS
WHERE EVENT_OBJECT_SCHEMA = ? AND EVENT_OBJECT_TABLE = ?
ORDER BY TRIGGER_NAME`, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.TableTriggerItem{}
	for rows.Next() {
		var name, timing, event string
		var body sql.NullString
		if err := rows.Scan(&name, &timing, &event, &body); err != nil {
			return nil, err
		}
		items = append(items, model.TableTriggerItem{
			Name:       name,
			Events:     []string{strings.TrimSpace(timing + " " + event)},
			Definition: optionalText(body.String),
		})
	}
	return items, rows.Err()
}

// referentialAction renders "NO ACTION" as "NO_ACTION", the form the UI already shows for
// SQL Server.
func referentialAction(rule string) *string {
	return optionalText(strings.ReplaceAll(strings.TrimSpace(rule), " ", "_"))
}
