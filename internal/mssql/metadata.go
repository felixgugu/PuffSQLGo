package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"puffsql/internal/model"
)

func (s *Server) GetDatabases(ctx context.Context) ([]model.DatabaseItem, error) {
	rows, err := s.query(ctx, nil, "SELECT name FROM sys.databases WHERE state = 0 ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.DatabaseItem{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, model.DatabaseItem{Name: name})
	}
	return items, rows.Err()
}

func (s *Server) GetTables(ctx context.Context, database, schema *string) ([]model.TableItem, error) {
	where := []string{"TABLE_TYPE IN ('BASE TABLE', 'VIEW')"}
	if schema != nil && strings.TrimSpace(*schema) != "" {
		where = append(where, "TABLE_SCHEMA = "+sqlString(*schema))
	}
	query := "SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE FROM INFORMATION_SCHEMA.TABLES WHERE " +
		strings.Join(where, " AND ") + " ORDER BY TABLE_SCHEMA, TABLE_NAME"
	rows, err := s.query(ctx, database, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.TableItem{}
	for rows.Next() {
		var item model.TableItem
		if err := rows.Scan(&item.Schema, &item.Name, &item.Kind); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) GetColumns(
	ctx context.Context,
	database *string,
	schema, table string,
) ([]model.ColumnItem, error) {
	sets, err := s.querySets(ctx, database, []string{columnsSelect(schema, table)})
	if err != nil {
		return nil, err
	}
	if len(sets) == 0 {
		return []model.ColumnItem{}, nil
	}
	return parseColumns(sets[0]), nil
}

func (s *Server) GetForeignKeys(
	ctx context.Context,
	database *string,
	schema, table *string,
) ([]model.ForeignKeyItem, error) {
	where := []string{"po.is_ms_shipped = 0"}
	if schema != nil && strings.TrimSpace(*schema) != "" {
		where = append(where, "SCHEMA_NAME(po.schema_id) = "+sqlString(*schema))
	}
	if table != nil && strings.TrimSpace(*table) != "" {
		where = append(where, "po.name = "+sqlString(*table))
	}
	query := `
SELECT fk.name, SCHEMA_NAME(po.schema_id), po.name, pc.name,
       SCHEMA_NAME(ro.schema_id), ro.name, rc.name
FROM sys.foreign_keys fk
JOIN sys.foreign_key_columns fkc ON fkc.constraint_object_id = fk.object_id
JOIN sys.objects po ON po.object_id = fk.parent_object_id
JOIN sys.columns pc ON pc.object_id = fkc.parent_object_id AND pc.column_id = fkc.parent_column_id
JOIN sys.objects ro ON ro.object_id = fk.referenced_object_id
JOIN sys.columns rc ON rc.object_id = fkc.referenced_object_id AND rc.column_id = fkc.referenced_column_id
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY fk.name, fkc.constraint_column_id`
	rows, err := s.query(ctx, database, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.ForeignKeyItem{}
	for rows.Next() {
		var item model.ForeignKeyItem
		if err := rows.Scan(
			&item.ConstraintName, &item.FromSchema, &item.FromTable, &item.FromColumn,
			&item.ToSchema, &item.ToTable, &item.ToColumn,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) GetDatabaseSchema(ctx context.Context, database *string) ([]model.TableSchema, error) {
	query := `
SELECT t.TABLE_SCHEMA, t.TABLE_NAME, t.TABLE_TYPE,
       c.COLUMN_NAME, c.DATA_TYPE,
       CAST(c.CHARACTER_MAXIMUM_LENGTH AS INT),
       CAST(c.NUMERIC_PRECISION AS INT), CAST(c.NUMERIC_SCALE AS INT),
       CASE WHEN c.IS_NULLABLE = 'YES' THEN 1 ELSE 0 END,
       CASE WHEN pk.COLUMN_NAME IS NULL THEN 0 ELSE 1 END,
       CASE WHEN EXISTS (
         SELECT 1 FROM sys.identity_columns ic
         JOIN sys.objects o ON o.object_id = ic.object_id
         WHERE SCHEMA_NAME(o.schema_id) = c.TABLE_SCHEMA
           AND o.name = c.TABLE_NAME AND ic.name = c.COLUMN_NAME
       ) THEN 1 ELSE 0 END,
       CASE WHEN EXISTS (
         SELECT 1 FROM sys.computed_columns cc
         JOIN sys.objects o ON o.object_id = cc.object_id
         WHERE SCHEMA_NAME(o.schema_id) = c.TABLE_SCHEMA
           AND o.name = c.TABLE_NAME AND cc.name = c.COLUMN_NAME
       ) THEN 1 ELSE 0 END,
       CASE WHEN c.DATA_TYPE IN ('timestamp', 'rowversion') THEN 1 ELSE 0 END,
       ISNULL(CAST((
         SELECT ep.value
         FROM sys.extended_properties ep
         WHERE ep.class = 1 AND ep.major_id = OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME))
           AND ep.minor_id = COLUMNPROPERTY(OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)), c.COLUMN_NAME, 'ColumnId')
           AND ep.name = N'MS_Description'
       ) AS nvarchar(4000)), N'')
FROM INFORMATION_SCHEMA.TABLES t
LEFT JOIN INFORMATION_SCHEMA.COLUMNS c
  ON t.TABLE_SCHEMA = c.TABLE_SCHEMA AND t.TABLE_NAME = c.TABLE_NAME
LEFT JOIN (
  SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
  FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
  JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
    ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
   AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
  WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
) pk ON c.TABLE_SCHEMA = pk.TABLE_SCHEMA
   AND c.TABLE_NAME = pk.TABLE_NAME AND c.COLUMN_NAME = pk.COLUMN_NAME
WHERE t.TABLE_TYPE IN ('BASE TABLE', 'VIEW')
ORDER BY t.TABLE_SCHEMA, t.TABLE_NAME, c.ORDINAL_POSITION`

	rows, err := s.query(ctx, database, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.TableSchema{}
	index := map[string]int{}
	for rows.Next() {
		var schema, name, kind, columnName, dataType sql.NullString
		var maxLength, precision, scale sql.NullInt64
		var nullable, primary, identity, computed, rowVersion sql.NullInt64
		var description sql.NullString
		if err := rows.Scan(
			&schema, &name, &kind, &columnName, &dataType,
			&maxLength, &precision, &scale, &nullable, &primary, &identity, &computed, &rowVersion,
			&description,
		); err != nil {
			return nil, err
		}
		key := schema.String + "\x00" + name.String
		tableIndex, exists := index[key]
		if !exists {
			tableIndex = len(items)
			index[key] = tableIndex
			items = append(items, model.TableSchema{
				Schema: schema.String, Name: name.String, Kind: kind.String,
				Columns: []model.ColumnItem{},
			})
		}
		if columnName.Valid && columnName.String != "" {
			items[tableIndex].Columns = append(items[tableIndex].Columns, model.ColumnItem{
				Name:         columnName.String,
				DataType:     dataType.String,
				MaxLength:    nullableInt(maxLength),
				Precision:    nullableInt(precision),
				Scale:        nullableInt(scale),
				IsNullable:   nullable.Int64 != 0,
				IsPrimaryKey: primary.Int64 != 0,
				IsIdentity:   identity.Int64 != 0,
				IsComputed:   computed.Int64 != 0,
				IsRowVersion: rowVersion.Int64 != 0,
				Description:  optionalText(description.String),
			})
		}
	}
	return items, rows.Err()
}

func (s *Server) SwitchDatabase(ctx context.Context, database string) error {
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "USE "+QuoteIdentifier(database)); err != nil {
		return err
	}
	s.SetDatabase(database)
	return nil
}

func (s *Server) query(ctx context.Context, database *string, statement string, args ...any) (*sql.Rows, error) {
	target := s.Database()
	if database != nil && strings.TrimSpace(*database) != "" {
		target = strings.TrimSpace(*database)
	}
	if target != "" {
		statement = "USE " + QuoteIdentifier(target) + ";\n" + statement
		s.SetDatabase(target)
	}
	return s.DB.QueryContext(ctx, statement, args...)
}

type rowSet struct {
	Columns []string
	Rows    [][]any
}

func (s *Server) querySets(ctx context.Context, database *string, statements []string) ([]rowSet, error) {
	rows, err := s.query(ctx, database, strings.Join(statements, "\n;\n"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sets []rowSet
	for {
		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		set := rowSet{Columns: columns, Rows: [][]any{}}
		values := make([]any, len(columns))
		scans := make([]any, len(columns))
		for i := range values {
			scans[i] = &values[i]
		}
		for rows.Next() {
			if err := rows.Scan(scans...); err != nil {
				return nil, err
			}
			row := make([]any, len(values))
			for i, value := range values {
				row[i] = normalizeValue(value, "")
			}
			set.Rows = append(set.Rows, row)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if len(columns) > 0 {
			sets = append(sets, set)
		}
		if !rows.NextResultSet() {
			break
		}
	}
	return sets, rows.Err()
}

func columnsSelect(schema, table string) string {
	return fmt.Sprintf(`
SELECT c.COLUMN_NAME, c.DATA_TYPE,
       CAST(c.CHARACTER_MAXIMUM_LENGTH AS INT),
       CAST(c.NUMERIC_PRECISION AS INT), CAST(c.NUMERIC_SCALE AS INT),
       CASE WHEN c.IS_NULLABLE = 'YES' THEN 1 ELSE 0 END,
       CASE WHEN pk.COLUMN_NAME IS NULL THEN 0 ELSE 1 END,
       CASE WHEN ic.object_id IS NULL THEN 0 ELSE 1 END,
       CASE WHEN cc.object_id IS NULL THEN 0 ELSE 1 END,
       CASE WHEN c.DATA_TYPE IN ('timestamp', 'rowversion') THEN 1 ELSE 0 END,
       ISNULL(dc.definition, N''), ISNULL(c.COLLATION_NAME, N''),
       ISNULL(cc.definition, N''), CAST(ISNULL(cc.is_persisted, 0) AS INT),
       CAST(ic.seed_value AS BIGINT), CAST(ic.increment_value AS BIGINT),
       ISNULL(CAST(ep.value AS nvarchar(4000)), N'')
FROM INFORMATION_SCHEMA.COLUMNS c
LEFT JOIN (
  SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
  FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
  JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
    ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
  WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
) pk ON c.TABLE_SCHEMA = pk.TABLE_SCHEMA AND c.TABLE_NAME = pk.TABLE_NAME
     AND c.COLUMN_NAME = pk.COLUMN_NAME
LEFT JOIN sys.objects so
  ON so.name = c.TABLE_NAME AND SCHEMA_NAME(so.schema_id) = c.TABLE_SCHEMA AND so.is_ms_shipped = 0
LEFT JOIN sys.columns sc ON sc.object_id = so.object_id AND sc.name = c.COLUMN_NAME
LEFT JOIN sys.default_constraints dc ON dc.object_id = sc.default_object_id
LEFT JOIN sys.computed_columns cc ON cc.object_id = sc.object_id AND cc.column_id = sc.column_id
LEFT JOIN sys.identity_columns ic ON ic.object_id = sc.object_id AND ic.column_id = sc.column_id
LEFT JOIN sys.extended_properties ep
  ON ep.class = 1 AND ep.major_id = so.object_id AND ep.minor_id = sc.column_id
 AND ep.name = N'MS_Description'
WHERE c.TABLE_SCHEMA = %s AND c.TABLE_NAME = %s
ORDER BY c.ORDINAL_POSITION`, sqlString(schema), sqlString(table))
}

func sqlString(value string) string {
	return "N'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case bool:
		if typed {
			return 1, true
		}
		return 0, true
	default:
		parsed, err := strconv.Atoi(asString(value))
		return parsed, err == nil
	}
}

func asInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	default:
		parsed, err := strconv.ParseInt(asString(value), 10, 64)
		return parsed, err == nil
	}
}

func asBool(value any) bool {
	number, ok := asInt(value)
	if ok {
		return number != 0
	}
	text := strings.ToLower(asString(value))
	return text == "1" || text == "true" || text == "yes"
}

func cell(row []any, index int) any {
	if index < 0 || index >= len(row) {
		return nil
	}
	return row[index]
}

func cellString(row []any, index int) string {
	return asString(cell(row, index))
}

func cellInt(row []any, index int) int {
	value, _ := asInt(cell(row, index))
	return value
}

func cellInt64(row []any, index int) int64 {
	value, _ := asInt64(cell(row, index))
	return value
}

func cellBool(row []any, index int) bool {
	return asBool(cell(row, index))
}

func nullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func optionalText(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func parseColumns(set rowSet) []model.ColumnItem {
	items := []model.ColumnItem{}
	for _, row := range set.Rows {
		name := cellString(row, 0)
		if name == "" {
			continue
		}
		isIdentity := cellBool(row, 7)
		item := model.ColumnItem{
			Name:               name,
			DataType:           cellString(row, 1),
			MaxLength:          nullableFromInt(cellInt(row, 2), !isNull(cell(row, 2))),
			Precision:          nullableFromInt(cellInt(row, 3), !isNull(cell(row, 3))),
			Scale:              nullableFromInt(cellInt(row, 4), !isNull(cell(row, 4))),
			IsNullable:         cellBool(row, 5),
			IsPrimaryKey:       cellBool(row, 6),
			IsIdentity:         isIdentity,
			IsComputed:         cellBool(row, 8),
			IsRowVersion:       cellBool(row, 9),
			DefaultDefinition:  optionalText(cellString(row, 10)),
			Collation:          optionalText(cellString(row, 11)),
			ComputedExpression: optionalText(cellString(row, 12)),
			IsPersisted:        cellBool(row, 13),
			Description:        optionalText(cellString(row, 16)),
		}
		if isIdentity {
			seed := cellInt64(row, 14)
			increment := cellInt64(row, 15)
			if isNull(cell(row, 14)) {
				seed = 1
			}
			if isNull(cell(row, 15)) {
				increment = 1
			}
			item.IdentitySeed = &seed
			item.IdentityIncrement = &increment
		}
		items = append(items, item)
	}
	return items
}

func nullableFromInt(value int, valid bool) *int {
	if !valid {
		return nil
	}
	return &value
}

func isNull(value any) bool {
	return value == nil || asString(value) == ""
}
