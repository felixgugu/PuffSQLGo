import type { CellValue } from '@/types/query';
import { mssqlDialect, type Dialect } from '@/dialects';
import {
  escapeIdentifier,
  formatCurrentDateTime,
  formatSqlLiteral,
  formatTableName,
} from './sqlGenerator';
import type { RowModification } from './batchUpdateGenerator';

/** One column/value pair of an INSERT. Identity / computed columns must already be removed. */
export interface InsertColumnValue {
  column: string;
  value: CellValue;
}

export interface InsertRowChange {
  values: InsertColumnValue[];
}

/** A row to delete, addressed by its complete primary key. */
export interface DeleteRowChange {
  pkWhere: Record<string, CellValue>;
}

export interface BatchChangeParams {
  tableName: string;
  schema?: string;
  database?: string;
  updates?: RowModification[];
  inserts?: InsertRowChange[];
  deletes?: DeleteRowChange[];
  /** Dialect of the target connection; SQL Server when omitted. */
  dialect?: Dialect;
  /**
   * Result column type names by column name. Lets a dialect write a numeric value that travelled
   * as text (BIGINT, DECIMAL) back as a number.
   */
  columnTypes?: Record<string, string>;
}

/** Everything the per-statement builders need to render one dialect. */
interface RenderContext {
  dialect: Dialect;
  fullTableName: string;
  typeOf: (column: string) => string | undefined;
}

/**
 * SQL Server verifies each statement touched exactly one row and throws otherwise. MySQL has no
 * equivalent inside a plain script, so keyed statements are capped with `LIMIT 1` instead.
 */
function rowGuard(ctx: RenderContext, code: number, message: string): string {
  if (ctx.dialect.engine === 'mysql') return '';
  return `\n\n  IF @@ROWCOUNT <> 1 THROW ${code}, '${message}', 1;`;
}

function limitOne(ctx: RenderContext): string {
  return ctx.dialect.engine === 'mysql' ? '\n  LIMIT 1' : '';
}

/**
 * Builds the `WHERE` clause for a primary key. Never falls back to "all columns": a DELETE must be
 * addressed by the key the caller verified, otherwise a missing value could widen the statement.
 */
function buildPkWhereClause(
  pkWhere: Record<string, CellValue>,
  context: string,
  ctx: RenderContext
): string {
  const entries = Object.entries(pkWhere);
  if (entries.length === 0) {
    throw new Error(`${context} lacks primary key condition`);
  }

  return entries
    .map(([name, value]) => {
      const column = escapeIdentifier(name, ctx.dialect);
      return value === null || value === undefined
        ? `${column} IS NULL`
        : `${column} = ${formatSqlLiteral(value, ctx.dialect, ctx.typeOf(name))}`;
    })
    .join('\n  AND ');
}

function buildDeleteStatement(
  change: DeleteRowChange,
  index: number,
  ctx: RenderContext
): string {
  const rowLabel = index + 1;
  return `-- [刪除 ${rowLabel}] 依主鍵刪除 1 列
  DELETE FROM ${ctx.fullTableName}
  WHERE ${buildPkWhereClause(change.pkWhere, `Delete change ${rowLabel}`, ctx)}${limitOne(ctx)};${rowGuard(ctx, 50003, `Delete change ${rowLabel} affected unexpected number of rows; transaction rolled back.`)}`;
}

function buildInsertStatement(
  change: InsertRowChange,
  index: number,
  ctx: RenderContext
): string {
  const rowLabel = index + 1;
  const seen = new Set<string>();
  for (const entry of change.values) {
    const key = entry.column.toLowerCase();
    if (seen.has(key)) {
      throw new Error(`Insert change ${rowLabel} contains duplicate column: ${entry.column}`);
    }
    seen.add(key);
  }

  const guard = rowGuard(
    ctx,
    50002,
    `Insert change ${rowLabel} affected unexpected number of rows; transaction rolled back.`
  );

  if (change.values.length === 0) {
    const defaults = ctx.dialect.engine === 'mysql' ? '() VALUES ()' : 'DEFAULT VALUES';
    return `-- [新增 ${rowLabel}] 以資料表預設值新增 1 列
  INSERT INTO ${ctx.fullTableName} ${defaults};${guard}`;
  }

  const columns = change.values
    .map((entry) => escapeIdentifier(entry.column, ctx.dialect))
    .join(', ');
  const values = change.values
    .map((entry) => formatSqlLiteral(entry.value, ctx.dialect, ctx.typeOf(entry.column)))
    .join(', ');

  return `-- [新增 ${rowLabel}] 寫入 ${change.values.length} 個欄位
  INSERT INTO ${ctx.fullTableName} (${columns})
  VALUES (${values});${guard}`;
}

function buildUpdateStatement(
  change: RowModification,
  ctx: RenderContext
): string | null {
  const updateEntries = Object.entries(change.updates);
  if (updateEntries.length === 0) return null;

  if (Object.keys(change.pkWhere).length === 0) {
    throw new Error(`Row modification at index ${change.rowIndex} lacks primary key condition`);
  }

  const setClauses = updateEntries.map(([column, value]) => {
    return `${escapeIdentifier(column, ctx.dialect)} = ${formatSqlLiteral(value, ctx.dialect, ctx.typeOf(column))}`;
  });

  const rowLabel = change.rowIndex + 1;
  return `-- [列 ${rowLabel}] 更新 ${updateEntries.length} 個欄位
  UPDATE ${ctx.fullTableName}
  SET ${setClauses.join(', ')}
  WHERE ${buildPkWhereClause(change.pkWhere, `Row modification at index ${rowLabel}`, ctx)}${limitOne(ctx)};${rowGuard(ctx, 50001, `Row update for index ${rowLabel} affected unexpected number of rows; transaction rolled back.`)}`;
}

/**
 * Generates one atomic script for pending grid changes.
 *
 * Order is DELETE → INSERT → UPDATE so a natural primary key freed by a deleted row can be reused
 * by an inserted one in the same transaction.
 *
 * SQL Server: every statement is guarded by `@@ROWCOUNT`, and any failure rolls the whole
 * transaction back inside the batch. MySQL: the statements run between `START TRANSACTION` and
 * `COMMIT`; the backend stops at the first failing statement and rolls the open transaction
 * back, and keyed statements carry `LIMIT 1`.
 */
export function generateBatchChangesScript(params: BatchChangeParams): string {
  const { tableName, schema, database } = params;
  const updates = params.updates ?? [];
  const inserts = params.inserts ?? [];
  const deletes = params.deletes ?? [];

  if (!updates.length && !inserts.length && !deletes.length) {
    throw new Error('No row changes provided to generate batch change script');
  }

  const dialect = params.dialect ?? mssqlDialect;
  const types = new Map<string, string>();
  for (const [column, type] of Object.entries(params.columnTypes ?? {})) {
    types.set(column.toLowerCase(), type);
  }
  const ctx: RenderContext = {
    dialect,
    fullTableName: formatTableName(tableName, schema, database, dialect),
    typeOf: (column) => types.get(column.toLowerCase()),
  };
  const timeHeader = `-- 自動產生批次異動 新增: ${inserts.length} 列, 更新: ${updates.length} 列, 刪除: ${deletes.length} 列, 時間: ${formatCurrentDateTime()}`;

  const statements: string[] = [];
  deletes.forEach((change, index) => {
    statements.push(buildDeleteStatement(change, index, ctx));
  });
  inserts.forEach((change, index) => {
    statements.push(buildInsertStatement(change, index, ctx));
  });
  updates.forEach((change) => {
    const statement = buildUpdateStatement(change, ctx);
    if (statement) statements.push(statement);
  });

  if (statements.length === 0) {
    throw new Error('No valid row changes found in changes');
  }

  if (dialect.engine === 'mysql') {
    return `${timeHeader}
START TRANSACTION;
  ${statements.join('\n\n  ')}

COMMIT;`;
  }

  return `${timeHeader}
IF @@TRANCOUNT <> 0 THROW 50000, 'Existing transaction detected; aborting execution.', 1;
BEGIN TRANSACTION;
BEGIN TRY
  ${statements.join('\n\n  ')}

  COMMIT TRANSACTION;
END TRY
BEGIN CATCH
  IF @@TRANCOUNT > 0 ROLLBACK TRANSACTION;
  THROW;
END CATCH;`;
}
