package mssql

import (
	"context"
	"fmt"
	"strings"

	"puffsql/internal/model"
)

func (s *Server) GetTableObjects(
	ctx context.Context,
	database *string,
	schema, table string,
) (model.TableObjects, error) {
	sets, err := s.querySets(ctx, database, tableObjectStatements(schema, table))
	if err != nil {
		return model.TableObjects{}, err
	}
	objects := model.TableObjects{
		Columns:     []model.ColumnItem{},
		Constraints: []model.TableConstraintItem{},
		Indexes:     []model.TableIndexItem{},
		Triggers:    []model.TableTriggerItem{},
	}
	if len(sets) > 0 {
		objects.Columns = parseColumns(sets[0])
	}
	objects.Constraints = parseConstraints(sets)
	if len(sets) > 5 {
		objects.Indexes = parseIndexes(sets[5])
	}
	if len(sets) > 6 {
		objects.Triggers = parseTriggers(sets[6])
	}
	return objects, nil
}

func tableObjectStatements(schema, table string) []string {
	schemaLiteral := sqlString(schema)
	tableLiteral := sqlString(table)
	return []string{
		columnsSelect(schema, table),
		fmt.Sprintf(`
SELECT kc.name, CASE kc.type WHEN 'PK' THEN 'PRIMARY_KEY' ELSE 'UNIQUE' END,
       CAST(i.is_disabled AS INT), CASE WHEN i.type = 1 THEN 1 ELSE 0 END,
       c.name, CAST(ISNULL(ic.is_descending_key, 0) AS INT)
FROM sys.key_constraints kc
JOIN sys.objects o ON o.object_id = kc.parent_object_id
JOIN sys.indexes i ON i.object_id = kc.parent_object_id AND i.index_id = kc.unique_index_id
LEFT JOIN sys.index_columns ic
  ON ic.object_id = i.object_id AND ic.index_id = i.index_id AND ic.is_included_column = 0
LEFT JOIN sys.columns c ON c.object_id = ic.object_id AND c.column_id = ic.column_id
WHERE SCHEMA_NAME(o.schema_id) = %s AND o.name = %s AND o.is_ms_shipped = 0
ORDER BY kc.name, ic.key_ordinal`, schemaLiteral, tableLiteral),
		fmt.Sprintf(`
SELECT cc.name, CAST(cc.is_disabled AS INT), CAST(cc.is_not_trusted AS INT),
       ISNULL(cc.definition, N'')
FROM sys.check_constraints cc
JOIN sys.objects o ON o.object_id = cc.parent_object_id
WHERE SCHEMA_NAME(o.schema_id) = %s AND o.name = %s AND o.is_ms_shipped = 0
ORDER BY cc.name`, schemaLiteral, tableLiteral),
		fmt.Sprintf(`
SELECT dc.name, ISNULL(dc.definition, N''), c.name
FROM sys.default_constraints dc
JOIN sys.objects o ON o.object_id = dc.parent_object_id
LEFT JOIN sys.columns c
  ON c.object_id = dc.parent_object_id AND c.column_id = dc.parent_column_id
WHERE SCHEMA_NAME(o.schema_id) = %s AND o.name = %s AND o.is_ms_shipped = 0
ORDER BY dc.name`, schemaLiteral, tableLiteral),
		fmt.Sprintf(`
SELECT fk.name, CAST(fk.is_disabled AS INT), CAST(fk.is_not_trusted AS INT),
       fk.delete_referential_action_desc, fk.update_referential_action_desc,
       SCHEMA_NAME(ro.schema_id), ro.name, pc.name, rc.name
FROM sys.foreign_keys fk
JOIN sys.foreign_key_columns fkc ON fkc.constraint_object_id = fk.object_id
JOIN sys.objects po ON po.object_id = fk.parent_object_id
JOIN sys.columns pc
  ON pc.object_id = fkc.parent_object_id AND pc.column_id = fkc.parent_column_id
JOIN sys.objects ro ON ro.object_id = fk.referenced_object_id
JOIN sys.columns rc
  ON rc.object_id = fkc.referenced_object_id AND rc.column_id = fkc.referenced_column_id
WHERE SCHEMA_NAME(po.schema_id) = %s AND po.name = %s AND po.is_ms_shipped = 0
ORDER BY fk.name, fkc.constraint_column_id`, schemaLiteral, tableLiteral),
		fmt.Sprintf(`
SELECT i.name, i.type_desc, CAST(i.is_unique AS INT), CAST(i.is_primary_key AS INT),
       CAST(i.is_unique_constraint AS INT), CAST(i.is_disabled AS INT),
       ISNULL(i.filter_definition, N''), c.name,
       CAST(ISNULL(ic.is_descending_key, 0) AS INT),
       CAST(ISNULL(ic.is_included_column, 0) AS INT)
FROM sys.indexes i
JOIN sys.objects o ON o.object_id = i.object_id
LEFT JOIN sys.index_columns ic
  ON ic.object_id = i.object_id AND ic.index_id = i.index_id
LEFT JOIN sys.columns c ON c.object_id = ic.object_id AND c.column_id = ic.column_id
WHERE SCHEMA_NAME(o.schema_id) = %s AND o.name = %s
  AND o.is_ms_shipped = 0 AND i.name IS NOT NULL
ORDER BY i.name, ic.is_included_column, ic.key_ordinal, ic.index_column_id`,
			schemaLiteral, tableLiteral),
		fmt.Sprintf(`
SELECT tr.name, CAST(tr.is_instead_of_trigger AS INT), CAST(tr.is_disabled AS INT),
       ISNULL(m.definition, N''),
       ISNULL(STUFF((
         SELECT N', ' + te.type_desc FROM sys.trigger_events te
         WHERE te.object_id = tr.object_id ORDER BY te.type
         FOR XML PATH(''), TYPE
       ).value('.', 'NVARCHAR(MAX)'), 1, 2, N''), N'')
FROM sys.triggers tr
JOIN sys.objects o ON o.object_id = tr.parent_id
LEFT JOIN sys.sql_modules m ON m.object_id = tr.object_id
WHERE SCHEMA_NAME(o.schema_id) = %s AND o.name = %s AND tr.is_ms_shipped = 0
ORDER BY tr.name`, schemaLiteral, tableLiteral),
	}
}

func parseConstraints(sets []rowSet) []model.TableConstraintItem {
	items := []model.TableConstraintItem{}
	index := map[string]int{}
	add := func(item model.TableConstraintItem) *model.TableConstraintItem {
		if existing, ok := index[item.Name]; ok {
			return &items[existing]
		}
		index[item.Name] = len(items)
		items = append(items, item)
		return &items[len(items)-1]
	}

	if len(sets) > 1 {
		for _, row := range sets[1].Rows {
			name := cellString(row, 0)
			if name == "" {
				continue
			}
			clustered := cellBool(row, 3)
			item := add(model.TableConstraintItem{
				Name: name, Kind: cellString(row, 1),
				KeyColumns:  []model.KeyColumnRef{},
				IsClustered: &clustered, IsDisabled: cellBool(row, 2),
				RefColumns: []string{},
			})
			if column := cellString(row, 4); column != "" {
				item.KeyColumns = append(item.KeyColumns, model.KeyColumnRef{
					Name: column, Descending: cellBool(row, 5),
				})
			}
			// Because items can grow, refresh by name after append.
			items[index[name]] = *item
		}
	}
	if len(sets) > 2 {
		for _, row := range sets[2].Rows {
			name := cellString(row, 0)
			if name == "" {
				continue
			}
			if _, exists := index[name]; !exists {
				add(model.TableConstraintItem{
					Name: name, Kind: "CHECK",
					KeyColumns: []model.KeyColumnRef{},
					Definition: optionalText(cellString(row, 3)),
					IsDisabled: cellBool(row, 1), IsNotTrusted: cellBool(row, 2),
					RefColumns: []string{},
				})
			}
		}
	}
	if len(sets) > 3 {
		for _, row := range sets[3].Rows {
			name := cellString(row, 0)
			if name == "" {
				continue
			}
			if _, exists := index[name]; !exists {
				add(model.TableConstraintItem{
					Name: name, Kind: "DEFAULT",
					KeyColumns:    []model.KeyColumnRef{},
					Definition:    optionalText(cellString(row, 1)),
					DefaultColumn: optionalText(cellString(row, 2)),
					RefColumns:    []string{},
				})
			}
		}
	}
	if len(sets) > 4 {
		for _, row := range sets[4].Rows {
			name := cellString(row, 0)
			if name == "" {
				continue
			}
			item := add(model.TableConstraintItem{
				Name: name, Kind: "FOREIGN_KEY",
				KeyColumns: []model.KeyColumnRef{},
				IsDisabled: cellBool(row, 1), IsNotTrusted: cellBool(row, 2),
				RefSchema:  optionalText(cellString(row, 5)),
				RefTable:   optionalText(cellString(row, 6)),
				RefColumns: []string{},
				OnDelete:   optionalText(cellString(row, 3)),
				OnUpdate:   optionalText(cellString(row, 4)),
			})
			if column := cellString(row, 7); column != "" {
				item.KeyColumns = append(item.KeyColumns, model.KeyColumnRef{Name: column})
			}
			if column := cellString(row, 8); column != "" {
				item.RefColumns = append(item.RefColumns, column)
			}
			items[index[name]] = *item
		}
	}
	return items
}

func parseIndexes(set rowSet) []model.TableIndexItem {
	items := []model.TableIndexItem{}
	index := map[string]int{}
	for _, row := range set.Rows {
		name := cellString(row, 0)
		if name == "" {
			continue
		}
		position, exists := index[name]
		if !exists {
			position = len(items)
			index[name] = position
			items = append(items, model.TableIndexItem{
				Name: name, TypeDesc: cellString(row, 1),
				IsUnique: cellBool(row, 2), IsPrimaryKey: cellBool(row, 3),
				IsUniqueConstraint: cellBool(row, 4), IsDisabled: cellBool(row, 5),
				FilterDefinition: optionalText(cellString(row, 6)),
				Columns:          []model.IndexColumnRef{},
			})
		}
		if column := cellString(row, 7); column != "" {
			items[position].Columns = append(items[position].Columns, model.IndexColumnRef{
				Name: column, Descending: cellBool(row, 8), IsIncluded: cellBool(row, 9),
			})
		}
	}
	return items
}

func parseTriggers(set rowSet) []model.TableTriggerItem {
	items := []model.TableTriggerItem{}
	for _, row := range set.Rows {
		name := cellString(row, 0)
		if name == "" {
			continue
		}
		events := []string{}
		for _, event := range strings.Split(cellString(row, 4), ",") {
			if event = strings.TrimSpace(event); event != "" {
				events = append(events, event)
			}
		}
		items = append(items, model.TableTriggerItem{
			Name: name, IsInsteadOf: cellBool(row, 1), IsDisabled: cellBool(row, 2),
			Definition: optionalText(cellString(row, 3)), Events: events,
		})
	}
	return items
}
