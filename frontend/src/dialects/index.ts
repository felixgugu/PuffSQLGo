import { mssqlDialect } from './mssql';
import { mysqlDialect } from './mysql';
import type { DatabaseEngine, Dialect } from './types';

export type { DatabaseEngine, Dialect, DialectCapabilities, RoutineKind } from './types';
export { mssqlDialect, mysqlDialect };

export const DIALECTS: Dialect[] = [mssqlDialect, mysqlDialect];

/** Resolves an engine name to its dialect. Unknown or missing engines are SQL Server. */
export function getDialect(engine?: string | null): Dialect {
  return engine === 'mysql' ? mysqlDialect : mssqlDialect;
}

export function isDatabaseEngine(value: unknown): value is DatabaseEngine {
  return value === 'mssql' || value === 'mysql';
}
