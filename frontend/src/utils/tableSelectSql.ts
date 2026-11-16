import { currentDialect, type Dialect } from '@/dialects';
import { wrapIdentifierIfNeeded } from './sqlParser';

/**
 * Builds the plain single-line SELECT statement the app generates for a table.
 *
 * - One line, and no TOP clause: a silent row cap hides the fact that the query did not
 *   return everything, and the statement is a starting point for the user to edit.
 * - No database prefix: the statement runs against the database the tab is bound to.
 * - Identifiers stay bare unless they actually need quoting (spaces, special characters
 *   or reserved words), which is the same treatment the Explorer applies to column names.
 * - Engines without a schema level (MySQL) get the bare table name: the "schema" the backend
 *   reports there is the database, which the tab is already bound to.
 */
export function generateTableSelectSql(
  schema: string | null | undefined,
  table: string,
  dialect: Dialect = currentDialect()
): string {
  return `SELECT * FROM ${tableTarget(schema, table, dialect)};\n`;
}

function tableTarget(schema: string | null | undefined, table: string, dialect: Dialect): string {
  const parts: string[] = [];
  const cleanSchema = (schema || '').trim();
  if (cleanSchema && dialect.showSchema) {
    parts.push(wrapIdentifierIfNeeded(cleanSchema, dialect));
  }
  parts.push(wrapIdentifierIfNeeded(table.trim(), dialect));
  return parts.join('.');
}

/**
 * Builds the single-column SELECT the quick finder offers for a column hit.
 * Same identifier treatment and no TOP clause as `generateTableSelectSql`.
 */
export function generateColumnSelectSql(
  schema: string | null | undefined,
  table: string,
  column: string,
  dialect: Dialect = currentDialect()
): string {
  const cleanColumn = wrapIdentifierIfNeeded(column.trim(), dialect);
  return `SELECT ${cleanColumn} FROM ${tableTarget(schema, table, dialect)};\n`;
}
