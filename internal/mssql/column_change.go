package mssql

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"puffsql/internal/model"
)

var commonColumnTypes = map[string]bool{
	"bit": true, "tinyint": true, "smallint": true, "int": true, "bigint": true,
	"decimal": true, "numeric": true, "money": true, "smallmoney": true,
	"float": true, "real": true, "date": true, "time": true, "datetime": true,
	"datetime2": true, "smalldatetime": true, "datetimeoffset": true,
	"char": true, "varchar": true, "nchar": true, "nvarchar": true,
	"binary": true, "varbinary": true, "uniqueidentifier": true, "xml": true,
}

func (s *Server) PreviewColumnChange(
	ctx context.Context,
	req model.ColumnChangeRequest,
) (model.ColumnChangePreview, error) {
	preview := model.ColumnChangePreview{
		Warnings:       []model.ColumnChangeIssue{},
		BlockingIssues: []model.ColumnChangeIssue{},
		Dependencies:   []model.ColumnDependency{},
	}
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	tableName, err := identifier(req.Table)
	if err != nil || tableName == "" {
		return preview, fmt.Errorf("invalid table name")
	}
	schema, err := identifier(req.Schema)
	if err != nil || schema == "" {
		return preview, fmt.Errorf("invalid schema name")
	}
	columnName, err := identifier(req.ColumnName)
	if err != nil || columnName == "" {
		return preview, fmt.Errorf("invalid column name")
	}
	req.Schema, req.Table, req.ColumnName = schema, tableName, columnName
	description, descriptionErr := normalizedDescription(req.Description)
	if descriptionErr != nil {
		preview.BlockingIssues = append(preview.BlockingIssues, issue("invalid_description", descriptionErr.Error()))
		preview.Fingerprint = columnPreviewFingerprint(preview)
		return preview, nil
	}
	qualified := QualifiedTable(schema, tableName)

	columns, err := s.GetColumns(ctx, &req.Database, schema, tableName)
	if err != nil {
		return preview, err
	}
	objects, err := s.GetTableObjects(ctx, &req.Database, schema, tableName)
	if err != nil {
		return preview, err
	}
	findColumn := func(name string) *model.ColumnItem {
		for i := range columns {
			if strings.EqualFold(columns[i].Name, name) {
				return &columns[i]
			}
		}
		return nil
	}

	switch op {
	case "add":
		if findColumn(columnName) != nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("duplicate_column", "欄位名稱已存在"))
			break
		}
		typeSQL, typeErr := validatedColumnType(req)
		if typeErr != nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("invalid_type", typeErr.Error()))
			break
		}
		defaultExpr, defaultErr := cleanDefaultExpression(req.DefaultExpression)
		if defaultErr != nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("invalid_default", defaultErr.Error()))
			break
		}
		hasRows, err := s.columnChangeHasRows(ctx, &req.Database, qualified)
		if err != nil {
			return preview, err
		}
		if hasRows {
			preview.AffectedRows = 1
		}
		if hasRows && !req.IsNullable && defaultExpr == "" {
			preview.BlockingIssues = append(preview.BlockingIssues, issue(
				"not_null_without_default", "資料表已有資料；新增 NOT NULL 欄位時必須提供預設值",
			))
		}
		nullability := columnNullability(req.IsNullable)
		clause := fmt.Sprintf("ALTER TABLE %s ADD %s %s %s", qualified, QuoteIdentifier(columnName), typeSQL, nullability)
		if defaultExpr != "" {
			constraint := generatedDefaultName(objects.Constraints, schema, tableName, columnName)
			clause += fmt.Sprintf(" CONSTRAINT %s DEFAULT %s", QuoteIdentifier(constraint), defaultExpr)
		}
		statements := []string{clause}
		statements = append(statements, descriptionStatements(schema, tableName, columnName, nil, description)...)
		preview.SQL = strings.Join(statements, ";\n") + ";\nGO"

	case "alter":
		original, nameErr := identifier(req.OriginalName)
		if nameErr != nil || original == "" {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("original_required", "缺少要修改的原始欄位名稱"))
			break
		}
		req.OriginalName = original
		current := findColumn(original)
		if current == nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("column_missing", "找不到原始欄位"))
			break
		}
		if current.IsIdentity || current.IsComputed || current.IsRowVersion {
			preview.BlockingIssues = append(preview.BlockingIssues, issue(
				"unsupported_column", "Identity、Computed 與 rowversion 欄位不支援此修改流程",
			))
			break
		}
		if !strings.EqualFold(original, columnName) && findColumn(columnName) != nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("rename_conflict", "新欄位名稱已存在"))
			break
		}
		typeSQL, typeErr := validatedColumnType(req)
		if typeErr != nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("invalid_type", typeErr.Error()))
			break
		}
		defaultExpr, defaultErr := cleanDefaultExpression(req.DefaultExpression)
		if defaultErr != nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("invalid_default", defaultErr.Error()))
			break
		}
		if !req.IsNullable {
			nulls, err := s.columnChangeNullCount(ctx, &req.Database, qualified, original)
			if err != nil {
				return preview, err
			}
			if nulls > 0 {
				preview.BlockingIssues = append(preview.BlockingIssues, issue(
					"null_values", fmt.Sprintf("欄位仍有 %d 筆 NULL，不能改成 NOT NULL", nulls),
				))
			}
		}
		if tooLong, maxLength, err := s.columnChangeOversizedValue(ctx, &req.Database, qualified, original, req); err != nil {
			return preview, err
		} else if tooLong {
			preview.BlockingIssues = append(preview.BlockingIssues, issue(
				"length_overflow", fmt.Sprintf("現有資料長度 %d 超過新欄位限制", maxLength),
			))
		}
		statements := []string{}
		for _, constraint := range objects.Constraints {
			if constraint.Kind == "DEFAULT" && constraint.DefaultColumn != nil &&
				strings.EqualFold(*constraint.DefaultColumn, original) {
				statements = append(statements, fmt.Sprintf(
					"ALTER TABLE %s DROP CONSTRAINT %s", qualified, QuoteIdentifier(constraint.Name),
				))
			}
		}
		if !strings.EqualFold(original, columnName) {
			statements = append(statements, fmt.Sprintf(
				"EXEC sp_rename N'%s', N'%s', N'COLUMN'",
				sqlLiteral(QualifiedTable(schema, tableName)+"."+QuoteIdentifier(original)),
				sqlLiteral(columnName),
			))
		}
		statements = append(statements, fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s %s %s",
			qualified, QuoteIdentifier(columnName), typeSQL, columnNullability(req.IsNullable),
		))
		if defaultExpr != "" {
			constraint := generatedDefaultName(objects.Constraints, schema, tableName, columnName)
			statements = append(statements, fmt.Sprintf(
				"ALTER TABLE %s ADD CONSTRAINT %s DEFAULT %s FOR %s",
				qualified, QuoteIdentifier(constraint), defaultExpr, QuoteIdentifier(columnName),
			))
		}
		statements = append(statements, descriptionStatements(
			schema, tableName, columnName, current.Description, description,
		)...)
		preview.SQL = strings.Join(statements, ";\n") + ";\nGO"
		preview.Warnings = append(preview.Warnings, issue(
			"conversion_risk", "型別轉換由 SQL Server 執行；若資料不相容，整個交易會回滾",
		))
		preview.Dependencies = collectColumnDependencies(objects, original)
		modules, moduleErr := s.columnModuleDependencies(ctx, &req.Database, qualified, original)
		if moduleErr != nil {
			return preview, moduleErr
		}
		preview.Dependencies = append(preview.Dependencies, modules...)

	case "drop":
		original, nameErr := identifier(req.OriginalName)
		if nameErr != nil || original == "" {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("original_required", "缺少要刪除的欄位名稱"))
			break
		}
		if findColumn(original) == nil {
			preview.BlockingIssues = append(preview.BlockingIssues, issue("column_missing", "找不到要刪除的欄位"))
			break
		}
		preview.Dependencies = collectColumnDependencies(objects, original)
		modules, moduleErr := s.columnModuleDependencies(ctx, &req.Database, qualified, original)
		if moduleErr != nil {
			return preview, moduleErr
		}
		preview.Dependencies = append(preview.Dependencies, modules...)
		for _, dependency := range preview.Dependencies {
			preview.BlockingIssues = append(preview.BlockingIssues, issue(
				"column_dependency", fmt.Sprintf("%s %s 仍依賴此欄位", dependency.Kind, dependency.Name),
			))
		}
		preview.SQL = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;\nGO", qualified, QuoteIdentifier(original))
	default:
		return preview, fmt.Errorf("unsupported operation %q", req.Operation)
	}

	sort.SliceStable(preview.Dependencies, func(i, j int) bool {
		if preview.Dependencies[i].Kind != preview.Dependencies[j].Kind {
			return preview.Dependencies[i].Kind < preview.Dependencies[j].Kind
		}
		return preview.Dependencies[i].Name < preview.Dependencies[j].Name
	})
	preview.Fingerprint = columnPreviewFingerprint(preview)
	return preview, nil
}

func (s *Server) ApplyColumnChange(
	ctx context.Context,
	req model.ColumnChangeRequest,
	fingerprint string,
) (model.ColumnChangeResult, error) {
	started := time.Now()
	preview, err := s.PreviewColumnChange(ctx, req)
	if err != nil {
		return model.ColumnChangeResult{}, err
	}
	if len(preview.BlockingIssues) > 0 {
		return model.ColumnChangeResult{}, fmt.Errorf("%s", preview.BlockingIssues[0].Message)
	}
	if preview.Fingerprint != fingerprint {
		return model.ColumnChangeResult{}, fmt.Errorf("資料表結構已變更，請重新預覽")
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return model.ColumnChangeResult{}, err
	}
	defer conn.Close()
	if req.Database != "" {
		if _, err := conn.ExecContext(ctx, "USE "+QuoteIdentifier(req.Database)); err != nil {
			return model.ColumnChangeResult{}, err
		}
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return model.ColumnChangeResult{}, err
	}
	sql := strings.TrimSuffix(strings.TrimSuffix(preview.SQL, "GO"), ";")
	for _, statement := range strings.Split(sql, ";\n") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			_ = tx.Rollback()
			return model.ColumnChangeResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return model.ColumnChangeResult{}, err
	}
	s.SetDatabase(req.Database)
	return model.ColumnChangeResult{ExecutionTimeMS: elapsedMillis(started)}, nil
}

func validatedColumnType(req model.ColumnChangeRequest) (string, error) {
	dataType := strings.ToLower(strings.TrimSpace(req.DataType))
	if !commonColumnTypes[dataType] {
		return "", fmt.Errorf("unsupported data type %q", req.DataType)
	}
	switch dataType {
	case "char", "varchar", "nchar", "nvarchar", "binary", "varbinary":
		if req.Length == nil || (*req.Length != -1 && *req.Length <= 0) {
			return "", fmt.Errorf("%s requires a positive length or MAX", dataType)
		}
		length := strconv.Itoa(*req.Length)
		if *req.Length == -1 {
			length = "MAX"
		}
		return QuoteIdentifier(dataType) + "(" + length + ")", nil
	case "decimal", "numeric":
		precision := 18
		scale := 0
		if req.Precision != nil {
			precision = *req.Precision
		}
		if req.Scale != nil {
			scale = *req.Scale
		}
		if precision < 1 || precision > 38 || scale < 0 || scale > precision {
			return "", fmt.Errorf("decimal precision/scale is invalid")
		}
		return fmt.Sprintf("%s(%d, %d)", QuoteIdentifier(dataType), precision, scale), nil
	case "time", "datetime2", "datetimeoffset":
		scale := 7
		if req.Scale != nil {
			scale = *req.Scale
		}
		if scale < 0 || scale > 7 {
			return "", fmt.Errorf("%s scale must be between 0 and 7", dataType)
		}
		return fmt.Sprintf("%s(%d)", QuoteIdentifier(dataType), scale), nil
	default:
		return QuoteIdentifier(dataType), nil
	}
}

func identifier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 128 || strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("invalid identifier")
	}
	return value, nil
}

func sqlLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func columnNullability(nullable bool) string {
	if nullable {
		return "NULL"
	}
	return "NOT NULL"
}

func cleanDefaultExpression(value *string) (string, error) {
	if value == nil {
		return "", nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return "", nil
	}
	if strings.Contains(text, ";") || regexp.MustCompile(`(?im)^\s*GO\s*$`).MatchString(text) {
		return "", fmt.Errorf("預設值必須是單一 T-SQL expression")
	}
	return text, nil
}

func normalizedDescription(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil, nil
	}
	if len([]rune(text)) > 4000 {
		return nil, fmt.Errorf("欄位備註不可超過 4000 個字元")
	}
	return &text, nil
}

func descriptionStatements(
	schema, table, column string,
	current, desired *string,
) []string {
	same := (current == nil && desired == nil) ||
		(current != nil && desired != nil && *current == *desired)
	if same {
		return nil
	}
	levels := fmt.Sprintf(
		"@level0type=N'SCHEMA', @level0name=%s, @level1type=N'TABLE', @level1name=%s, @level2type=N'COLUMN', @level2name=%s",
		sqlString(schema), sqlString(table), sqlString(column),
	)
	switch {
	case current == nil && desired != nil:
		return []string{fmt.Sprintf(
			"EXEC sys.sp_addextendedproperty @name=N'MS_Description', @value=%s, %s",
			sqlString(*desired), levels,
		)}
	case current != nil && desired == nil:
		return []string{fmt.Sprintf(
			"EXEC sys.sp_dropextendedproperty @name=N'MS_Description', %s",
			levels,
		)}
	case current != nil && desired != nil:
		return []string{fmt.Sprintf(
			"EXEC sys.sp_updateextendedproperty @name=N'MS_Description', @value=%s, %s",
			sqlString(*desired), levels,
		)}
	default:
		return nil
	}
}

func issue(code, message string) model.ColumnChangeIssue {
	return model.ColumnChangeIssue{Code: code, Message: message}
}

func generatedDefaultName(existing []model.TableConstraintItem, schema, table, column string) string {
	base := "DF_" + sanitizeName(schema) + "_" + sanitizeName(table) + "_" + sanitizeName(column)
	if len(base) > 120 {
		base = base[:120]
	}
	used := map[string]bool{}
	for _, item := range existing {
		used[strings.ToLower(item.Name)] = true
	}
	name := base
	for i := 2; used[strings.ToLower(name)]; i++ {
		name = fmt.Sprintf("%s_%d", base, i)
	}
	return name
}

func sanitizeName(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func collectColumnDependencies(objects model.TableObjects, column string) []model.ColumnDependency {
	result := []model.ColumnDependency{}
	for _, c := range objects.Constraints {
		matches := false
		for _, key := range c.KeyColumns {
			matches = matches || strings.EqualFold(key.Name, column)
		}
		if c.DefaultColumn != nil && strings.EqualFold(*c.DefaultColumn, column) {
			matches = true
		}
		if c.Definition != nil && expressionMentionsColumn(*c.Definition, column) {
			matches = true
		}
		if matches {
			result = append(result, model.ColumnDependency{Kind: c.Kind, Name: c.Name, Detail: "constraint"})
		}
	}
	for _, index := range objects.Indexes {
		for _, c := range index.Columns {
			if strings.EqualFold(c.Name, column) {
				result = append(result, model.ColumnDependency{Kind: "INDEX", Name: index.Name, Detail: index.TypeDesc})
				break
			}
		}
	}
	for _, c := range objects.Columns {
		if c.ComputedExpression != nil && expressionMentionsColumn(*c.ComputedExpression, column) {
			result = append(result, model.ColumnDependency{Kind: "COMPUTED_COLUMN", Name: c.Name, Detail: "computed expression"})
		}
	}
	for _, trigger := range objects.Triggers {
		if trigger.Definition != nil && expressionMentionsColumn(*trigger.Definition, column) {
			result = append(result, model.ColumnDependency{Kind: "TRIGGER", Name: trigger.Name, Detail: "trigger definition"})
		}
	}
	return result
}

func expressionMentionsColumn(expression, column string) bool {
	pattern := `(?i)(^|[^a-zA-Z0-9_])` + regexp.QuoteMeta(column) + `([^a-zA-Z0-9_]|$)`
	return regexp.MustCompile(pattern).MatchString(expression)
}

func columnPreviewFingerprint(preview model.ColumnChangePreview) string {
	data, _ := json.Marshal(struct {
		SQL       string
		Warnings  []model.ColumnChangeIssue
		Blocking  []model.ColumnChangeIssue
		DependsOn []model.ColumnDependency
	}{preview.SQL, preview.Warnings, preview.BlockingIssues, preview.Dependencies})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Server) columnChangeHasRows(ctx context.Context, database *string, qualified string) (bool, error) {
	var exists int
	err := s.queryOne(ctx, database, "SELECT CASE WHEN EXISTS(SELECT 1 FROM "+qualified+") THEN 1 ELSE 0 END", &exists)
	return exists == 1, err
}

func (s *Server) columnChangeNullCount(ctx context.Context, database *string, qualified, column string) (int64, error) {
	var count int64
	err := s.queryOne(ctx, database,
		fmt.Sprintf("SELECT COUNT_BIG(*) FROM %s WHERE %s IS NULL", qualified, QuoteIdentifier(column)), &count)
	return count, err
}

func (s *Server) columnChangeOversizedValue(
	ctx context.Context,
	database *string,
	qualified, column string,
	req model.ColumnChangeRequest,
) (bool, int64, error) {
	dataType := strings.ToLower(req.DataType)
	var expression string
	var limit int64
	switch dataType {
	case "char", "varchar", "nchar", "nvarchar":
		if req.Length == nil || *req.Length < 0 {
			return false, 0, nil
		}
		limit = int64(*req.Length)
		expression = fmt.Sprintf("MAX(LEN(CONVERT(nvarchar(max), %s)))", QuoteIdentifier(column))
	case "binary", "varbinary":
		if req.Length == nil || *req.Length < 0 {
			return false, 0, nil
		}
		limit = int64(*req.Length)
		expression = fmt.Sprintf("MAX(DATALENGTH(%s))", QuoteIdentifier(column))
	default:
		return false, 0, nil
	}
	var maxLength int64
	err := s.queryOne(ctx, database,
		fmt.Sprintf("SELECT ISNULL(%s, 0) FROM %s", expression, qualified), &maxLength)
	return maxLength > limit, maxLength, err
}

func (s *Server) columnModuleDependencies(
	ctx context.Context,
	database *string,
	qualified, column string,
) ([]model.ColumnDependency, error) {
	statement := fmt.Sprintf(`
SELECT DISTINCT ISNULL(SCHEMA_NAME(o.schema_id), N''), o.name, o.type_desc
FROM sys.sql_expression_dependencies d
JOIN sys.objects o ON o.object_id = d.referencing_id
WHERE d.referenced_id = OBJECT_ID(%s)
  AND d.referenced_minor_id = COLUMNPROPERTY(OBJECT_ID(%s), %s, 'ColumnId')`,
		sqlString(qualified), sqlString(qualified), sqlString(column))
	rows, err := s.query(ctx, database, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.ColumnDependency{}
	for rows.Next() {
		var schema, name, kind string
		if err := rows.Scan(&schema, &name, &kind); err != nil {
			return nil, err
		}
		result = append(result, model.ColumnDependency{
			Kind: kind, Name: qualifiedNameText(schema, name), Detail: "SQL module dependency",
		})
	}
	return result, rows.Err()
}

func qualifiedNameText(schema, name string) string {
	if schema == "" {
		return name
	}
	return schema + "." + name
}
