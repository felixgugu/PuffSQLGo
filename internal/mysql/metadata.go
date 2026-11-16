package mysql

import (
	"context"
	"database/sql"
	"strings"

	"puffsql/internal/model"
)

// columnFields is the shared select list read by scanColumn.
const columnFields = `c.COLUMN_NAME, c.DATA_TYPE, c.CHARACTER_MAXIMUM_LENGTH, c.NUMERIC_PRECISION,
       c.NUMERIC_SCALE, c.DATETIME_PRECISION, c.IS_NULLABLE, c.COLUMN_KEY, c.EXTRA,
       c.COLUMN_DEFAULT, c.COLLATION_NAME, c.GENERATION_EXPRESSION, c.COLUMN_COMMENT`

func (s *Server) GetDatabases(ctx context.Context) ([]model.DatabaseItem, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT SCHEMA_NAME FROM information_schema.SCHEMATA ORDER BY SCHEMA_NAME")
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

// GetTables lists the tables and views of one database. The schema argument is accepted for
// interface compatibility; in MySQL it can only name the database itself.
func (s *Server) GetTables(ctx context.Context, database, schema *string) ([]model.TableItem, error) {
	target := s.resolveDatabase(database)
	if target == "" && schema != nil {
		target = strings.TrimSpace(*schema)
	}
	items := []model.TableItem{}
	if target == "" {
		return items, nil
	}
	// SYSTEM VIEW (information_schema) is reported as VIEW so those objects stay browsable.
	rows, err := s.DB.QueryContext(ctx, `
SELECT TABLE_SCHEMA, TABLE_NAME,
       CASE WHEN TABLE_TYPE = 'BASE TABLE' THEN 'BASE TABLE' ELSE 'VIEW' END
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = ?
ORDER BY TABLE_NAME`, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
	rows, err := s.DB.QueryContext(ctx, `
SELECT `+columnFields+`
FROM information_schema.COLUMNS c
WHERE c.TABLE_SCHEMA = ? AND c.TABLE_NAME = ?
ORDER BY c.ORDINAL_POSITION`, s.tableDatabase(database, schema), table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.ColumnItem{}
	for rows.Next() {
		var column columnRow
		if err := rows.Scan(column.targets()...); err != nil {
			return nil, err
		}
		items = append(items, column.item())
	}
	return items, rows.Err()
}

func (s *Server) GetDatabaseSchema(ctx context.Context, database *string) ([]model.TableSchema, error) {
	target := s.resolveDatabase(database)
	items := []model.TableSchema{}
	if target == "" {
		return items, nil
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT t.TABLE_SCHEMA, t.TABLE_NAME,
       CASE WHEN t.TABLE_TYPE = 'BASE TABLE' THEN 'BASE TABLE' ELSE 'VIEW' END,
       `+columnFields+`
FROM information_schema.TABLES t
LEFT JOIN information_schema.COLUMNS c
  ON c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
WHERE t.TABLE_SCHEMA = ?
ORDER BY t.TABLE_NAME, c.ORDINAL_POSITION`, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	index := map[string]int{}
	for rows.Next() {
		var schema, name, kind string
		var column columnRow
		targets := append([]any{&schema, &name, &kind}, column.targets()...)
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		position, exists := index[name]
		if !exists {
			position = len(items)
			index[name] = position
			items = append(items, model.TableSchema{
				Schema: schema, Name: name, Kind: kind, Columns: []model.ColumnItem{},
			})
		}
		if column.name.Valid && column.name.String != "" {
			items[position].Columns = append(items[position].Columns, column.item())
		}
	}
	return items, rows.Err()
}

func (s *Server) GetForeignKeys(
	ctx context.Context,
	database *string,
	schema, table *string,
) ([]model.ForeignKeyItem, error) {
	target := s.resolveDatabase(database)
	if target == "" && schema != nil {
		target = strings.TrimSpace(*schema)
	}
	query := `
SELECT CONSTRAINT_NAME, TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME,
       REFERENCED_TABLE_SCHEMA, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = ? AND REFERENCED_TABLE_NAME IS NOT NULL`
	args := []any{target}
	if table != nil && strings.TrimSpace(*table) != "" {
		query += " AND TABLE_NAME = ?"
		args = append(args, strings.TrimSpace(*table))
	}
	query += " ORDER BY TABLE_NAME, CONSTRAINT_NAME, ORDINAL_POSITION"
	rows, err := s.DB.QueryContext(ctx, query, args...)
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

// tableDatabase resolves the database a table lives in. Callers pass the database name in the
// schema argument (MySQL has no separate schema), so it is the fallback when database is empty.
func (s *Server) tableDatabase(database *string, schema string) string {
	if database != nil && strings.TrimSpace(*database) != "" {
		return strings.TrimSpace(*database)
	}
	if strings.TrimSpace(schema) != "" {
		return strings.TrimSpace(schema)
	}
	return s.Database()
}

// columnRow receives one information_schema.COLUMNS row selected with columnFields.
type columnRow struct {
	name, dataType                       sql.NullString
	maxLength, precision, scale, timePre sql.NullInt64
	nullable, key, extra                 sql.NullString
	defaultValue, collation, generation  sql.NullString
	comment                              sql.NullString
}

func (c *columnRow) targets() []any {
	return []any{
		&c.name, &c.dataType, &c.maxLength, &c.precision, &c.scale, &c.timePre,
		&c.nullable, &c.key, &c.extra, &c.defaultValue, &c.collation, &c.generation, &c.comment,
	}
}

func (c *columnRow) item() model.ColumnItem {
	extra := strings.ToLower(c.extra.String)
	item := model.ColumnItem{
		Name:         c.name.String,
		DataType:     c.dataType.String,
		MaxLength:    nullableInt(c.maxLength),
		Precision:    nullableInt(c.precision),
		Scale:        nullableInt(c.scale),
		IsNullable:   strings.EqualFold(c.nullable.String, "YES"),
		IsPrimaryKey: strings.EqualFold(c.key.String, "PRI"),
		IsIdentity:   strings.Contains(extra, "auto_increment"),
		IsComputed:   strings.Contains(extra, "generated") && strings.TrimSpace(c.generation.String) != "",
		IsPersisted:  strings.Contains(extra, "stored generated"),
		Collation:    optionalText(c.collation.String),
		Description:  optionalText(c.comment.String),
	}
	// Temporal types report their fractional-second precision as the scale.
	if item.Scale == nil && c.timePre.Valid && c.timePre.Int64 > 0 {
		item.Scale = nullableInt(c.timePre)
	}
	if item.IsComputed {
		item.ComputedExpression = optionalText(c.generation.String)
	} else if c.defaultValue.Valid {
		value := c.defaultValue.String
		item.DefaultDefinition = &value
	}
	return item
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
