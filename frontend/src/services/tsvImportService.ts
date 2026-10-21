import { invokeCommand, isWails } from './api';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { queryService } from './queryService';
import { buildTableUniqueKeysSql, parseUniqueKeys } from '@/utils/tsvImport';
import type {
  ImportCapabilities,
  ImportPayloadRow,
  ImportProgress,
  ImportResult,
  TsvImportTarget,
  UniqueKeyPlan,
} from '@/types/tsvImport';

export const IMPORT_PROGRESS_EVENT = 'sqlight:import-progress';

async function listenImportProgress(
  importId: string,
  onProgress: (progress: ImportProgress) => void
): Promise<() => void> {
  if (!isWails()) return () => {};
  try {
    return EventsOn(IMPORT_PROGRESS_EVENT, (payload: ImportProgress) => {
      if (!payload || payload.importId !== importId) return;
      onProgress(payload);
    });
  } catch (err) {
    console.warn('[PuffSQL] import progress listener unavailable:', err);
    return () => {};
  }
}

export const tsvImportService = {
  async getCapabilities(target: TsvImportTarget): Promise<ImportCapabilities> {
    return invokeCommand<ImportCapabilities>('get_table_import_capabilities', {
      connectionId: target.connId,
      database: target.database,
      schema: target.schema,
      table: target.table,
    });
  },

  async getUniqueKeys(target: TsvImportTarget): Promise<UniqueKeyPlan[]> {
    try {
      const result = await queryService.executeQuery(
        target.connId,
        target.database,
        buildTableUniqueKeysSql(target.schema, target.table)
      );
      return parseUniqueKeys(result.resultSets);
    } catch (err) {
      // Unique metadata is only used for in-file duplicate warnings; the database stays the
      // final authority through the transactional insert, so this never blocks the import.
      console.warn('[PuffSQL] unable to read unique index metadata:', err);
      return [];
    }
  },

  async importRows(options: {
    target: TsvImportTarget;
    columns: string[];
    rows: ImportPayloadRow[];
    manualIdentity: boolean;
    importId: string;
    onProgress?: (progress: ImportProgress) => void;
  }): Promise<ImportResult> {
    const { target, columns, rows, manualIdentity, importId, onProgress } = options;
    const unlisten = onProgress ? await listenImportProgress(importId, onProgress) : () => {};
    try {
      return await invokeCommand<ImportResult>('import_table_rows', {
        connectionId: target.connId,
        database: target.database,
        schema: target.schema,
        table: target.table,
        columns,
        rows,
        manualIdentity,
        importId,
      });
    } finally {
      unlisten();
    }
  },
};
