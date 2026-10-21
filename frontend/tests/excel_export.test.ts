import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import { readProjectSource, readSource } from './helpers/source';

describe('excel database export', () => {
  test('Wails and the service expose the workbook export', () => {
    assert.match(
      readProjectSource('frontend/wailsjs/go/main/App.d.ts'),
      /export function ExportDatabaseSchemaToExcel\(/
    );
    assert.match(
      readProjectSource('frontend/wailsjs/go/main/App.d.ts'),
      /export function ExportDatabaseTablesToExcel\(/
    );
    const api = readSource('src/services/api.ts');
    assert.match(api, /case 'export_database_schema_to_excel':/);
    assert.match(api, /case 'export_database_tables_to_excel':/);
    assert.match(api, /backend\.ExportDatabaseSchemaToExcel\(value\.connectionId, value\.database\)/);
    assert.match(api, /backend\.ExportDatabaseTablesToExcel\(/);
    const service = readSource('src/services/excelExportService.ts');
    assert.match(service, /puffsql:excel-export-progress/);
    assert.match(service, /EventsOn\(EXCEL_EXPORT_PROGRESS_EVENT/);
  });

  test('the database context menu exports only through the shared service', () => {
    const sidebar = readSource('src/components/layout/AppSidebar.vue');
    assert.match(sidebar, /exportSelectedTablesExcel/);
    assert.match(sidebar, /exportAllTableSchemasExcel/);
    assert.match(sidebar, /excelExportService\.exportTables\(/);
    assert.match(sidebar, /excelExportService\.exportSchema\(/);
    assert.match(sidebar, /<ExcelTableExportModal/);
    assert.doesNotMatch(sidebar, /SheetJS|exceljs/i);
  });
});
