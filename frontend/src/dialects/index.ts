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

/**
 * The dialect of the connection the UI is currently working against. The connection store keeps
 * it in sync with the active connection; SQL helpers in `utils/` use it as their default so a
 * caller only passes a dialect explicitly when it targets a different connection.
 */
let current: Dialect = mssqlDialect;

export function currentDialect(): Dialect {
  return current;
}

export function setCurrentDialect(dialect: Dialect): void {
  current = dialect;
}
