import type { Dialect } from './types';

function quote(name: string): string {
  return `\`${name.replace(/`/g, '``')}\``;
}

/** Driver type names of result columns that hold numbers (see go-sql-driver/mysql fields.go). */
const NUMERIC_TYPE = /^(UNSIGNED )?(TINYINT|SMALLINT|MEDIUMINT|INT|BIGINT)$|^(DECIMAL|FLOAT|DOUBLE|YEAR|BIT)$/i;
const PLAIN_NUMBER = /^-?\d+(\.\d+)?$/;

/** MySQL has no schema below the database: the "schema" the backend reports is the database. */
function databaseOf(database: string | undefined, schema: string | undefined): string {
  return (database || '').trim() || (schema || '').trim();
}

export const mysqlDialect: Dialect = {
  engine: 'mysql',
  label: 'MySQL',
  shortLabel: 'MySQL',
  sqlName: 'MySQL SQL',
  protocol: 'MySQL protocol',

  defaultPort: 3306,
  defaultUsername: 'root',
  defaultDatabase: '',
  fallbackDatabase: '',
  systemDatabases: ['information_schema', 'mysql', 'performance_schema', 'sys'],

  showSchema: false,
  defaultSchema: '',
  usesGoBatches: false,
  mysqlLexing: true,
  identifierQuote: { open: '`', close: '`' },
  formatterLanguage: 'mysql',
  sessionLabel: 'Connection ID',

  capabilities: {
    executionPlan: false,
    statistics: false,
    dbaQueries: false,
    builtinTemplates: false,
    columnEditor: false,
    structureViewer: true,
    objectScripts: false,
    tsvImport: false,
    schemaCsvExport: false,
  },

  quoteIdentifier: quote,

  // Backslash is an escape character in MySQL string literals under the default sql_mode.
  stringLiteral(text) {
    return `'${text.replace(/\\/g, '\\\\').replace(/'/g, "''")}'`;
  },

  // MySQL compares a string with a number as floating point, which is inexact for BIGINT and
  // DECIMAL values. A numeric value that travelled as text (to keep its precision) must be
  // written back as a bare number.
  bareNumericText(text, dataType) {
    return Boolean(dataType) && NUMERIC_TYPE.test(dataType!.trim()) && PLAIN_NUMBER.test(text);
  },

  qualifyTable(table, schema, database) {
    const owner = databaseOf(database, schema);
    return owner ? `${quote(owner)}.${quote(table.trim())}` : quote(table.trim());
  },

  selectLimitSql(qualifiedTable, limit) {
    return `SELECT * FROM ${qualifiedTable} LIMIT ${limit};`;
  },

  // COLUMN_TYPE already carries the full declaration ("decimal(38,10)", "bigint unsigned").
  structureSql(database, schema, table) {
    const owner = this.stringLiteral(databaseOf(database, schema));
    return `
SELECT
    c.ORDINAL_POSITION AS Ordinal,
    c.COLUMN_NAME AS ColumnName,
    c.DATA_TYPE AS DataType,
    c.COLUMN_TYPE AS FullType,
    c.CHARACTER_MAXIMUM_LENGTH AS MaxLength,
    c.NUMERIC_PRECISION AS NumericPrecision,
    c.NUMERIC_SCALE AS NumericScale,
    CASE WHEN c.IS_NULLABLE = 'YES' THEN 'YES' ELSE 'NO' END AS IsNullable,
    CASE WHEN c.COLUMN_KEY = 'PRI' THEN 1 ELSE 0 END AS IsPrimaryKey,
    CASE WHEN c.EXTRA LIKE '%auto_increment%' THEN 1 ELSE 0 END AS IsIdentity,
    c.COLUMN_DEFAULT AS DefaultValue,
    c.COLLATION_NAME AS Collation,
    c.COLUMN_COMMENT AS Description
FROM information_schema.COLUMNS c
WHERE c.TABLE_SCHEMA = ${owner} AND c.TABLE_NAME = ${this.stringLiteral(table)}
ORDER BY c.ORDINAL_POSITION;
    `.trim();
  },

  routineListSql(database) {
    const literal = this.stringLiteral(database);
    return `SELECT ROUTINE_SCHEMA, ROUTINE_NAME, ROUTINE_TYPE FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ${literal} ORDER BY ROUTINE_NAME;`;
  },

  definitionSql(database, schema, name, kind) {
    return `SHOW CREATE ${kind} ${this.qualifyTable(name, schema, database)};`;
  },

  callProcedureSql(database, schema, name) {
    return `-- 執行預存程序 ${name}\n-- TODO: 如有參數請填入括號內\nCALL ${this.qualifyTable(name, schema, database)}();\n`;
  },

  callFunctionSql(database, schema, name) {
    return `-- 呼叫自訂函數 ${name}\n-- TODO: 如有參數請填入括號內\nSELECT ${this.qualifyTable(name, schema, database)}() AS \`Result\`;\n`;
  },
};
