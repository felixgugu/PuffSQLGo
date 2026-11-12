import type { CellValue } from '@/types/query';
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
}

/**
 * Builds the `WHERE` clause for a primary key. Never falls back to "all columns": a DELETE must be
 * addressed by the key the caller verified, otherwise a missing value could widen the statement.
 */
function buildPkWhereClause(pkWhere: Record<string, CellValue>, context: string): string {
  const entries = Object.entries(pkWhere);
  if (entries.length === 0) {
    throw new Error(`${context} lacks primary key condition`);
  }

  return entries
    .map(([name, value]) => {
      const column = escapeIdentifier(name);
      return value === null || value === undefined
        ? `${column} IS NULL`
        : `${column} = ${formatSqlLiteral(value)}`;
    })
    .join('\n  AND ');
}

function buildDeleteStatement(
  change: DeleteRowChange,
  index: number,
  fullTableName: string
): string {
  const rowLabel = index + 1;
  return `-- [刪除 ${rowLabel}] 依主鍵刪除 1 列
  DELETE FROM ${fullTableName}
  WHERE ${buildPkWhereClause(change.pkWhere, `Delete change ${rowLabel}`)};

  IF @@ROWCOUNT <> 1 THROW 50003, 'Delete change ${rowLabel} affected unexpected number of rows; transaction rolled back.', 1;`;
}

function buildInsertStatement(
  change: InsertRowChange,
  index: number,
  fullTableName: string
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

  if (change.values.length === 0) {
    return `-- [新增 ${rowLabel}] 以資料表預設值新增 1 列
  INSERT INTO ${fullTableName} DEFAULT VALUES;

  IF @@ROWCOUNT <> 1 THROW 50002, 'Insert change ${rowLabel} affected unexpected number of rows; transaction rolled back.', 1;`;
  }

  const columns = change.values.map((entry) => escapeIdentifier(entry.column)).join(', ');
  const values = change.values.map((entry) => formatSqlLiteral(entry.value)).join(', ');

  return `-- [新增 ${rowLabel}] 寫入 ${change.values.length} 個欄位
  INSERT INTO ${fullTableName} (${columns})
  VALUES (${values});

  IF @@ROWCOUNT <> 1 THROW 50002, 'Insert change ${rowLabel} affected unexpected number of rows; transaction rolled back.', 1;`;
}

function buildUpdateStatement(
  change: RowModification,
  fullTableName: string
): string | null {
  const updateEntries = Object.entries(change.updates);
  if (updateEntries.length === 0) return null;

  if (Object.keys(change.pkWhere).length === 0) {
    throw new Error(`Row modification at index ${change.rowIndex} lacks primary key condition`);
  }

  const setClauses = updateEntries.map(([column, value]) => {
    return `${escapeIdentifier(column)} = ${formatSqlLiteral(value)}`;
  });

  const rowLabel = change.rowIndex + 1;
  return `-- [列 ${rowLabel}] 更新 ${updateEntries.length} 個欄位
  UPDATE ${fullTableName}
  SET ${setClauses.join(', ')}
  WHERE ${buildPkWhereClause(change.pkWhere, `Row modification at index ${rowLabel}`)};

  IF @@ROWCOUNT <> 1 THROW 50001, 'Row update for index ${rowLabel} affected unexpected number of rows; transaction rolled back.', 1;`;
}

/**
 * Generates one atomic T-SQL batch for pending grid changes.
 *
 * Order is DELETE → INSERT → UPDATE so a natural primary key freed by a deleted row can be reused
 * by an inserted one in the same transaction. Every statement is guarded by `@@ROWCOUNT`, and any
 * failure rolls the whole transaction back.
 */
export function generateBatchChangesScript(params: BatchChangeParams): string {
  const { tableName, schema, database } = params;
  const updates = params.updates ?? [];
  const inserts = params.inserts ?? [];
  const deletes = params.deletes ?? [];

  if (!updates.length && !inserts.length && !deletes.length) {
    throw new Error('No row changes provided to generate batch change script');
  }

  const fullTableName = formatTableName(tableName, schema, database);
  const timeHeader = `-- 自動產生批次異動 新增: ${inserts.length} 列, 更新: ${updates.length} 列, 刪除: ${deletes.length} 列, 時間: ${formatCurrentDateTime()}`;

  const statements: string[] = [];
  deletes.forEach((change, index) => {
    statements.push(buildDeleteStatement(change, index, fullTableName));
  });
  inserts.forEach((change, index) => {
    statements.push(buildInsertStatement(change, index, fullTableName));
  });
  updates.forEach((change) => {
    const statement = buildUpdateStatement(change, fullTableName);
    if (statement) statements.push(statement);
  });

  if (statements.length === 0) {
    throw new Error('No valid row changes found in changes');
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
