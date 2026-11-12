import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import { generateBatchChangesScript } from '../src/utils/batchChangeGenerator';
import { generateBatchUpdateScript } from '../src/utils/batchUpdateGenerator';
import { editCell } from '../src/utils/tabulatorGrid';
import { readSource } from './helpers/source';

describe('batch_change_generator', () => {
  test('mixed changes run in one guarded transaction, DELETE before INSERT before UPDATE', () => {
    const script = generateBatchChangesScript({
      tableName: 'Users',
      schema: 'dbo',
      database: 'MyDb',
      deletes: [{ pkWhere: { id: 1 } }],
      inserts: [
        {
          values: [
            { column: 'id', value: 2 },
            { column: 'username', value: 'alice' },
          ],
        },
      ],
      updates: [
        {
          rowIndex: 1,
          updates: { email: null },
          pkWhere: { id: 3 },
        },
      ],
    });

    assert.match(script, /BEGIN TRANSACTION;/);
    assert.match(script, /COMMIT TRANSACTION;/);
    assert.match(script, /ROLLBACK TRANSACTION;/);
    assert.match(script, /\[MyDb\]\.\[dbo\]\.\[Users\]/);
    assert.match(script, /DELETE FROM/);
    assert.match(script, /INSERT INTO/);
    assert.match(script, /UPDATE/);
    assert.match(script, /WHERE \[id\] = 1/);
    assert.match(script, /\[username\]\)\s*\n\s*VALUES \(2, N'alice'\)/);
    assert.match(script, /\[email\] = NULL/);
    assert.match(script, /IF @@ROWCOUNT <> 1 THROW 50001/);
    assert.match(script, /IF @@ROWCOUNT <> 1 THROW 50002/);
    assert.match(script, /IF @@ROWCOUNT <> 1 THROW 50003/);

    const deleteAt = script.indexOf('DELETE FROM');
    const insertAt = script.indexOf('INSERT INTO');
    const updateAt = script.indexOf('UPDATE ');
    assert.ok(deleteAt !== -1 && insertAt !== -1 && updateAt !== -1);
    assert.ok(deleteAt < insertAt, 'DELETE must run before INSERT so a natural key can be reused');
    assert.ok(insertAt < updateAt, 'INSERT must run before UPDATE');
  });

  test('an insert without writable columns falls back to DEFAULT VALUES', () => {
    const script = generateBatchChangesScript({
      tableName: 'Audit',
      inserts: [{ values: [] }],
    });

    assert.match(script, /INSERT INTO \[dbo\]\.\[Audit\] DEFAULT VALUES;/);
    assert.match(script, /IF @@ROWCOUNT <> 1 THROW 50002/);
  });

  test('identifiers and string literals are escaped', () => {
    const script = generateBatchChangesScript({
      tableName: 'Weird]Table',
      schema: 'My Schema',
      inserts: [{ values: [{ column: 'na]me', value: "O'Brien" }] }],
    });

    assert.match(script, /\[My Schema\]\.\[Weird\]\]Table\]/);
    assert.match(script, /\[na\]\]me\]/);
    assert.match(script, /N'O''Brien'/);
  });

  test('empty plans and deletes without a primary key are rejected', () => {
    assert.throws(() => generateBatchChangesScript({ tableName: 'Users' }));
    assert.throws(() =>
      generateBatchChangesScript({
        tableName: 'Users',
        deletes: [{ pkWhere: {} }],
      })
    );
    assert.throws(() =>
      generateBatchChangesScript({
        tableName: 'Users',
        updates: [{ rowIndex: 0, updates: {}, pkWhere: { id: 1 } }],
      })
    );
  });

  test('the legacy update-only entry point delegates to the same builder', () => {
    const script = generateBatchUpdateScript({
      tableName: 'Users',
      modifications: [{ rowIndex: 0, updates: { username: 'bob' }, pkWhere: { id: 7 } }],
    });

    assert.match(script, /UPDATE \[dbo\]\.\[Users\]/);
    assert.match(script, /WHERE \[id\] = 7/);
    assert.doesNotMatch(script, /INSERT INTO|DELETE FROM/);
  });
});

describe('result_grid_f2_edit', () => {
  const source = readSource('src/components/results/ResultGridItem.vue');

  test('F2 edits the active range cell, mirroring Tabulator Enter-to-edit', () => {
    assert.match(source, /@keydown="handleGridKeydown"/);
    assert.match(source, /event\.key !== 'F2'/);
    assert.match(source, /activeRangeCell\(grid\.table\.value\) \?\? lastClickedCell/);
    assert.match(source, /event\.preventDefault\(\)/);
    assert.match(source, /editCell\(grid\.table\.value, cell\)/);
  });

  test('editCell reaches the internal cell through Tabulator edit module', () => {
    const internal = { id: 'internal-cell' };
    const component = { _getSelf: () => internal } as any;
    const calls: unknown[] = [];
    const table = {
      modules: { edit: { editCell: (cell: unknown) => calls.push(cell) } },
    } as any;

    assert.equal(editCell(table, component), true);
    assert.deepEqual(calls, [internal]);
    assert.equal(editCell(null, component), false);
    assert.equal(editCell(table, null), false);
  });
});

describe('data_view_layout', () => {
  const source = readSource('src/components/modals/DataViewModal.vue');

  test('the field table uses compact rows', () => {
    assert.doesNotMatch(source, /py-1\.5 px-3\.5/, 'the data view rows must be compact');
    assert.ok(
      (source.match(/py-1 px-3\.5/g) ?? []).length >= 8,
      'header and body cells must all use the compact padding'
    );
    assert.ok(
      (source.match(/leading-snug/g) ?? []).length >= 2,
      'value and property cells must use the tighter line height'
    );
  });

  test('the value column carries a light grey band without covering its grid lines', () => {
    assert.match(source, /<colgroup>/);
    assert.match(
      source,
      /<col class="bg-dark-950 dark:bg-dark-800" \/>/,
      'the value tint must live on the column, not on the header/cell background'
    );
    assert.doesNotMatch(
      source,
      /<th class="[^"]*bg-dark-950/,
      'the value header must share the same chrome colour as every other header'
    );
    assert.match(source, /bg-emerald-500\/10 text-ok/);
    assert.match(source, /bg-rose-500\/10 text-danger/);
    assert.match(source, /bg-dark-800\/50 text-dark-300/);
    assert.match(source, /bg-dark-900\/70 dark:bg-dark-900\/60/);
    assert.match(source, /hover:bg-dark-750\/30 dark:hover:bg-dark-800\/50/);
    assert.match(source, /divide-y divide-dark-700/);
    assert.match(source, /border-r border-dark-700/);
  });
});

describe('result_grid_row_editing', () => {
  const source = readSource('src/components/results/ResultGridItem.vue');

  test('the toolbar offers insert and delete next to revert and commit', () => {
    assert.match(source, /@click="handleAddRow"/);
    assert.match(source, /@click="handleDeleteRows"/);
    assert.match(source, /:disabled="!canAddRow"/);
    assert.match(source, /:disabled="!canDeleteRows"/);
    assert.match(source, /:disabled="pendingRowCount === 0"/);
  });

  test('adding a row works from an empty result, not only from the mounted grid', () => {
    assert.match(source, /const canAddRow = computed\([\s\S]{0,160}?editability\.value\.canEdit/);
    assert.doesNotMatch(
      source.slice(source.indexOf('const canAddRow'), source.indexOf('const canDeleteRows')),
      /hasRows/,
      'canAddRow must not require existing rows'
    );
    assert.match(source, /:label="\$t\('results\.addFirstRow'\)"/);
    assert.match(source, /v-if="!resultSet \|\| !hasRows"/);
    assert.match(source, /await grid\.sync\(\)/);
  });

  test('structural row changes are tracked and rendered as pending inserts / deletes', () => {
    assert.match(source, /const insertedRows = shallowRef<Set<CellValue\[\]>>\(new Set\(\)\)/);
    assert.match(source, /const deletedRows = shallowRef<Set<CellValue\[\]>>\(new Set\(\)\)/);
    assert.match(source, /const rowRevision = ref\(0\)/);
    assert.match(source, /sqlight-row-inserted/);
    assert.match(source, /sqlight-row-pending-delete/);
    assert.match(source, /element\.classList\.toggle\('sqlight-row-inserted'/);
    assert.match(source, /element\.classList\.toggle\('sqlight-row-pending-delete'/);
  });

  test('the context menu can delete an existing row and restore a pending insert', () => {
    assert.match(source, /@click="toggleContextRowDeletion"/);
    assert.match(source, /results\.undoDeleteRow/);
    assert.match(source, /results\.removeInsertedRow/);
  });

  test('duplicating a row copies values but clears keys and generated columns', () => {
    assert.match(source, /@click="duplicateContextRow"/);
    assert.match(source, /results\.duplicateRow/);

    const body = source.slice(
      source.indexOf('async function handleDuplicateRow'),
      source.indexOf('async function duplicateContextRow')
    );
    assert.match(body, /generated\.has\(name\) \|\| primaryKeys\.has\(name\)/);
    assert.match(body, /isBinaryCellValue\(value\) \? null : value/);
    assert.match(body, /sourceIndex \+ 1/);
    assert.match(body, /splice\(insertAt, 0, row\)/);
  });

  test('commits carry inserts, updates and deletes into the batch generator', () => {
    assert.match(source, /generateBatchChangesScript\(\{/);
    assert.match(source, /updates,/);
    assert.match(source, /inserts,/);
    assert.match(source, /deletes,/);
    assert.match(source, /commitModal\.insertCount = inserts\.length/);
    assert.match(source, /commitModal\.deleteCount = deletes\.length/);
  });

  test('a SQL error returned in the query result is not reported as a successful commit', () => {
    assert.match(source, /result\.messages\?\.find\(\(message\) => message\.level === 'error'\)/);
    assert.match(source, /reportCommitError\(sqlError\.message, direct\)/);
    assert.match(source, /message\.message\.toLowerCase\(\)\.includes\('cancel'\)/);
    assert.match(source, /t\('results\.commitFailed', \{ error: message \}\)/);
  });

  test('pending row state survives Tabulator lookups and is painted explicitly', () => {
    assert.match(source, /__sqlightRowState/);
    assert.match(source, /function isPendingInsert/);
    assert.match(source, /function isPendingDelete/);
    assert.match(source, /function applyRowStateClasses/);
    assert.match(source, /element\.classList\.toggle\('sqlight-row-pending-delete', isPendingDelete\(data\)\)/);
  });

  test('inserted rows omit identity, computed and defaulted columns', () => {
    const body = source.slice(
      source.indexOf('function buildInsertValues'),
      source.indexOf('function validateInsertRows')
    );
    assert.match(body, /schemaColumn\.isIdentity \|\| schemaColumn\.isComputed \|\| schemaColumn\.isRowVersion/);
    assert.match(body, /schemaColumn\.defaultDefinition/);
  });
});
