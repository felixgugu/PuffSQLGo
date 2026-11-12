import type { CellValue } from '@/types/query';
import { generateBatchChangesScript } from './batchChangeGenerator';

export interface RowModification {
  rowIndex: number;
  /** Keyed by column name, value is the new CellValue to update */
  updates: Record<string, CellValue>;
  /** Keyed by PK column name, value is the original PK value used for WHERE condition */
  pkWhere: Record<string, CellValue>;
}

export interface BatchUpdateParams {
  tableName: string;
  schema?: string;
  database?: string;
  modifications: RowModification[];
}

/**
 * Generates an atomic, safe T-SQL batch script to commit multiple row/cell updates.
 * Kept as the update-only entry point; the shared implementation also covers INSERT / DELETE.
 */
export function generateBatchUpdateScript(params: BatchUpdateParams): string {
  return generateBatchChangesScript({
    tableName: params.tableName,
    schema: params.schema,
    database: params.database,
    updates: params.modifications,
  });
}
