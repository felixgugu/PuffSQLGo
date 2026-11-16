import type { Dialect } from './types';

function quote(name: string): string {
  return `[${name.replace(/\]/g, ']]')}]`;
}

/**
 * Column catalog for the structure viewer. Both dialects return the same column aliases so the
 * viewer maps the rows without knowing the engine.
 */
function structureSql(schema: string, table: string): string {
  const escapedSchema = schema.replace(/'/g, "''");
  const escapedTable = table.replace(/'/g, "''");
  return `
SELECT
    c.ORDINAL_POSITION AS Ordinal,
    c.COLUMN_NAME AS ColumnName,
    c.DATA_TYPE AS DataType,
    CASE
        WHEN c.DATA_TYPE IN ('nchar', 'nvarchar') THEN
            c.DATA_TYPE + '(' + CASE WHEN c.CHARACTER_MAXIMUM_LENGTH = -1 THEN 'MAX' ELSE CAST(c.CHARACTER_MAXIMUM_LENGTH AS VARCHAR(10)) END + ')'
        WHEN c.DATA_TYPE IN ('char', 'varchar', 'binary', 'varbinary') THEN
            c.DATA_TYPE + '(' + CASE WHEN c.CHARACTER_MAXIMUM_LENGTH = -1 THEN 'MAX' ELSE CAST(c.CHARACTER_MAXIMUM_LENGTH AS VARCHAR(10)) END + ')'
        WHEN c.DATA_TYPE IN ('decimal', 'numeric') THEN
            c.DATA_TYPE + '(' + CAST(c.NUMERIC_PRECISION AS VARCHAR(10)) + ', ' + CAST(c.NUMERIC_SCALE AS VARCHAR(10)) + ')'
        WHEN c.DATA_TYPE IN ('datetime2', 'time', 'datetimeoffset') THEN
            c.DATA_TYPE + '(' + CAST(c.DATETIME_PRECISION AS VARCHAR(10)) + ')'
        ELSE c.DATA_TYPE
    END AS FullType,
    c.CHARACTER_MAXIMUM_LENGTH AS MaxLength,
    c.NUMERIC_PRECISION AS NumericPrecision,
    c.NUMERIC_SCALE AS NumericScale,
    CASE WHEN c.IS_NULLABLE = 'YES' THEN 'YES' ELSE 'NO' END AS IsNullable,
    CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END AS IsPrimaryKey,
    ISNULL(COLUMNPROPERTY(OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)), c.COLUMN_NAME, 'IsIdentity'), 0) AS IsIdentity,
    c.COLUMN_DEFAULT AS DefaultValue,
    c.COLLATION_NAME AS Collation,
    ISNULL(CAST((
      SELECT ep.value
      FROM sys.extended_properties ep
      WHERE ep.class = 1
        AND ep.major_id = OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME))
        AND ep.minor_id = COLUMNPROPERTY(OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)), c.COLUMN_NAME, 'ColumnId')
        AND ep.name = N'MS_Description'
    ) AS nvarchar(4000)), N'') AS Description
FROM INFORMATION_SCHEMA.COLUMNS c
LEFT JOIN (
    SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
    FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
    JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
        ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
        AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
    WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
) pk ON c.TABLE_SCHEMA = pk.TABLE_SCHEMA
    AND c.TABLE_NAME = pk.TABLE_NAME
    AND c.COLUMN_NAME = pk.COLUMN_NAME
WHERE c.TABLE_SCHEMA = '${escapedSchema}' AND c.TABLE_NAME = '${escapedTable}'
ORDER BY c.ORDINAL_POSITION;
    `.trim();
}

export const mssqlDialect: Dialect = {
  engine: 'mssql',
  label: 'Microsoft SQL Server',
  shortLabel: 'SQL Server',
  sqlName: 'T-SQL',
  protocol: 'TDS',

  defaultPort: 1433,
  defaultUsername: 'sa',
  defaultDatabase: 'master',
  fallbackDatabase: 'master',
  systemDatabases: ['master', 'tempdb', 'model', 'msdb'],

  showSchema: true,
  defaultSchema: 'dbo',
  usesGoBatches: true,
  mysqlLexing: false,
  identifierQuote: { open: '[', close: ']' },
  formatterLanguage: 'tsql',
  sessionLabel: 'SPID',

  capabilities: {
    executionPlan: true,
    statistics: true,
    dbaQueries: true,
    builtinTemplates: true,
    columnEditor: true,
    structureViewer: true,
    objectScripts: true,
    tsvImport: true,
    schemaCsvExport: true,
  },

  quoteIdentifier: quote,

  stringLiteral(text) {
    return `N'${text.replace(/'/g, "''")}'`;
  },

  // SQL Server converts a quoted number to the column type exactly, so text stays quoted.
  bareNumericText() {
    return false;
  },

  qualifyTable(table, schema, database) {
    const cleanSchema = (schema || 'dbo').trim();
    const parts: string[] = [];
    if (database && database.trim()) parts.push(quote(database.trim()));
    if (cleanSchema) parts.push(quote(cleanSchema));
    parts.push(quote(table.trim()));
    return parts.join('.');
  },

  selectLimitSql(qualifiedTable, limit) {
    return `SELECT TOP ${limit} * FROM ${qualifiedTable};`;
  },

  structureSql(_database, schema, table) {
    return structureSql(schema, table);
  },

  routineListSql(database) {
    return `SELECT ROUTINE_SCHEMA, ROUTINE_NAME, ROUTINE_TYPE FROM [${database}].INFORMATION_SCHEMA.ROUTINES ORDER BY ROUTINE_SCHEMA, ROUTINE_NAME;`;
  },

  definitionSql(database, schema, name) {
    return `USE [${database}]; SELECT OBJECT_DEFINITION(OBJECT_ID(N'[${schema}].[${name}]')) AS [def];`;
  },

  callProcedureSql(database, schema, name) {
    return `-- 執行預存程序 [${schema}].[${name}]\nUSE [${database}];\nGO\n\nEXEC [${schema}].[${name}];\n`;
  },

  callFunctionSql(database, schema, name) {
    return `-- 呼叫自訂函數 [${schema}].[${name}]\nUSE [${database}];\nGO\n\nSELECT [${schema}].[${name}]();\n`;
  },
};
