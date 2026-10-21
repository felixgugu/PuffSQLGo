import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import { readProjectSource, readSource } from './helpers/source';

describe('column editor', () => {
  test('Wails exposes preview and apply as separate operations', () => {
    const bindings = readProjectSource('frontend/wailsjs/go/main/App.d.ts');
    assert.match(bindings, /export function PreviewColumnChange\(/);
    assert.match(bindings, /export function ApplyColumnChange\(/);

    const api = readSource('src/services/api.ts');
    assert.match(api, /case 'preview_column_change':/);
    assert.match(api, /case 'apply_column_change':/);
    assert.match(api, /backend\.PreviewColumnChange\(value\.req\)/);
    assert.match(api, /backend\.ApplyColumnChange\(value\.req, value\.fingerprint\)/);
  });

  test('the dialog cannot execute before previewing and blocks unsafe changes', () => {
    const source = readSource('src/components/modals/ColumnEditorModal.vue');
    assert.match(source, /await columnChangeService\.preview\(buildRequest\(\)\)/);
    assert.match(
      source,
      /await columnChangeService\.apply\(buildRequest\(\), preview\.value\.fingerprint\)/,
      'apply must use the fingerprint returned by preview'
    );
    assert.match(source, /preview\.blockingIssues\.length/, 'blocking issues are rendered');
    assert.match(
      source,
      /:disabled="!!preview\?\.blockingIssues\.length"/,
      'blocked previews cannot execute'
    );
    assert.match(source, /<SqlCodeViewer :code="preview\.sql"/, 'the actual SQL is previewed');
    assert.match(source, /v-model="form\.description"/, 'column descriptions are editable');
    assert.match(source, /description: form\.description\.trim\(\) \|\| null/);
  });

  test('both table structure and Explorer open the shared editor', () => {
    const structure = readSource('src/components/editor/TableStructureViewer.vue');
    const sidebar = readSource('src/components/layout/AppSidebar.vue');
    for (const source of [structure, sidebar]) {
      assert.match(source, /<ColumnEditorModal/);
      assert.match(source, /puffsql:schema-changed/);
    }
    assert.match(structure, /openColumnEditor\('add'\)/);
    assert.match(sidebar, /openColumnEditorFromContext\('add'\)/);
    assert.match(sidebar, /openColumnEditorFromContext\('alter'/);
    assert.match(sidebar, /openColumnEditorFromContext\('drop'/);
  });
});
