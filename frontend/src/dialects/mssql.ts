import type { Dialect } from './types';

function quote(name: string): string {
  return `[${name.replace(/\]/g, ']]')}]`;
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
