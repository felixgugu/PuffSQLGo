export type DatabaseEngine = 'mssql' | 'mysql';

/** Features that only exist for some engines. The UI hides the entry points of a disabled one. */
export interface DialectCapabilities {
  /** Estimated/actual execution plan (SHOWPLAN / STATISTICS XML) and the plan viewer. */
  executionPlan: boolean;
  /** STATISTICS IO/TIME and the per-query performance telemetry wrapper. */
  statistics: boolean;
  /** The built-in DBA query menu (DMV based). */
  dbaQueries: boolean;
  /** Built-in SQL templates and table inspection templates (custom templates always show). */
  builtinTemplates: boolean;
  /** Add / alter / drop column through the column editor and the ALTER templates. */
  columnEditor: boolean;
  /** The table structure viewer. */
  structureViewer: boolean;
  /** Generated index / constraint / trigger maintenance scripts. */
  objectScripts: boolean;
  /** TSV import into a table. */
  tsvImport: boolean;
  /** Schema export to CSV (catalog queries built in the frontend). */
  schemaCsvExport: boolean;
}

export type RoutineKind = 'PROCEDURE' | 'FUNCTION';

/**
 * Everything the frontend needs to know about one SQL dialect. Dialect objects are pure: they
 * hold no connection state and are safe to use from `utils/`.
 */
export interface Dialect {
  engine: DatabaseEngine;
  /** Product name shown in the UI, e.g. "Microsoft SQL Server". */
  label: string;
  /** Short name for tight spaces and AI prompts, e.g. "SQL Server". */
  shortLabel: string;
  /** Name of the SQL language for prompts and copy, e.g. "T-SQL". */
  sqlName: string;
  /** Wire protocol shown in the connection dialog header. */
  protocol: string;

  defaultPort: number;
  defaultUsername: string;
  /** Initial database of a new profile ("" when the engine does not need one). */
  defaultDatabase: string;
  /** Database used when nothing else is known ("" = none). */
  fallbackDatabase: string;
  systemDatabases: string[];

  /** False when the engine has no schema level below the database. */
  showSchema: boolean;
  defaultSchema: string;
  /** Whether a line holding only `GO` separates batches. */
  usesGoBatches: boolean;
  /** Whether `#` starts a line comment and `\` escapes inside string literals. */
  mysqlLexing: boolean;
  /** Characters that open and close a quoted identifier. */
  identifierQuote: { open: string; close: string };
  /** `language` option of the sql-formatter package. */
  formatterLanguage: 'tsql' | 'mysql';
  /** Label of the server session id shown in the status bar. */
  sessionLabel: string;

  capabilities: DialectCapabilities;

  quoteIdentifier(name: string): string;
  /** Text literal including its quotes. */
  stringLiteral(text: string): string;
  /**
   * Whether a value that arrived as text should be written as a bare numeric literal for a
   * column of this data type (the driver type name of the result column).
   */
  bareNumericText(text: string, dataType: string | undefined): boolean;
  /** Fully qualified, quoted table name. */
  qualifyTable(table: string, schema?: string, database?: string): string;
  /** `SELECT *` over an already quoted table name, capped at `limit` rows. */
  selectLimitSql(qualifiedTable: string, limit: number): string;

  /** Query returning (schema, name, type) rows for every procedure and function. */
  routineListSql(database: string): string;
  /** Query whose first cell holds (or, for MySQL, whose row contains) the object's source. */
  definitionSql(database: string, schema: string, name: string, kind: RoutineKind | 'VIEW' | 'TABLE'): string;
  /** Template that calls a stored procedure. */
  callProcedureSql(database: string, schema: string, name: string): string;
  /** Template that calls a scalar function. */
  callFunctionSql(database: string, schema: string, name: string): string;
}
