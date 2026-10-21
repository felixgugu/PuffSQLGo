export interface ExcelExportProgress {
  current: number;
  total: number;
  table: string;
  sheets: number;
  rows: number;
  columns: number;
}

export interface ExcelExportResult {
  canceled: boolean;
  filePath: string;
  tableCount: number;
  sheetCount: number;
  columnCount: number;
  totalRows: number;
  truncatedCount: number;
  executionTimeMs: number;
}

export interface ExcelTableRef {
  schema: string;
  name: string;
}
