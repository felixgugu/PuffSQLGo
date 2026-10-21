import { invokeCommand, isWails } from './api';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import type {
  ExcelExportProgress,
  ExcelExportResult,
  ExcelTableRef,
} from '@/types/excelExport';

export const EXCEL_EXPORT_PROGRESS_EVENT = 'puffsql:excel-export-progress';

export const excelExportService = {
  async exportTables(
    connectionId: string,
    database: string,
    tables: ExcelTableRef[],
    onProgress?: (progress: ExcelExportProgress) => void
  ): Promise<ExcelExportResult> {
    return withProgress(onProgress, () =>
      invokeCommand<ExcelExportResult>('export_database_tables_to_excel', {
        connectionId, database, tables,
      })
    );
  },

  async exportSchema(
    connectionId: string,
    database: string,
    onProgress?: (progress: ExcelExportProgress) => void
  ): Promise<ExcelExportResult> {
    return withProgress(onProgress, () =>
      invokeCommand<ExcelExportResult>('export_database_schema_to_excel', {
        connectionId, database,
      })
    );
  },
};

async function withProgress(
  onProgress: ((progress: ExcelExportProgress) => void) | undefined,
  work: () => Promise<ExcelExportResult>
): Promise<ExcelExportResult> {
  const stop = onProgress && isWails()
    ? EventsOn(EXCEL_EXPORT_PROGRESS_EVENT, (progress: ExcelExportProgress) => onProgress(progress))
    : () => {};
  try {
    return await work();
  } finally {
    stop();
  }
}
