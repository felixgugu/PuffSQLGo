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
    structureViewer: false,
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
