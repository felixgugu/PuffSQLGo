import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import {
  buildExplorerFilterIndex,
  explorerDatabaseKey,
  explorerRoutineKey,
  explorerTableKey,
  matchingColumns,
} from '../src/utils/explorerFilter';
import type { ColumnItem, RoutineItem, TableItem, TableSchema } from '../src/types/schema';

function column(name: string): ColumnItem {
  return {
    name,
    dataType: 'int',
    isNullable: false,
    isPrimaryKey: false,
    isIdentity: false,
  };
}

describe('explorer_filter', () => {
  const connections = [
    { id: 'c1', name: 'Local', databases: ['AppDb', 'Archive'] },
    { id: 'c2', name: 'Remote', databases: ['Warehouse'] },
  ];

  const tablesByDb: Record<string, TableItem[]> = {
    'c1:AppDb': [
      { schema: 'dbo', name: 'Orders', kind: 'BASE TABLE' },
      { schema: 'dbo', name: 'Customers', kind: 'BASE TABLE' },
      { schema: 'dbo', name: 'vw_OpenOrders', kind: 'VIEW' },
    ],
    'c1:Archive': [{ schema: 'dbo', name: 'OrdersHistory', kind: 'BASE TABLE' }],
    'c2:Warehouse': [{ schema: 'fact', name: 'Sales', kind: 'BASE TABLE' }],
  };

  const routinesByDb: Record<string, RoutineItem[]> = {
    'c1:AppDb': [{ schema: 'dbo', name: 'usp_Report', kind: 'PROCEDURE' }],
  };

  const schemasByDb: Record<string, TableSchema[]> = {
    'c1:AppDb': [
      {
        schema: 'dbo',
        name: 'Orders',
        kind: 'BASE TABLE',
        columns: [column('OrderId'), column('CustomerId'), column('CreatedAt')],
      },
      {
        schema: 'dbo',
        name: 'Customers',
        kind: 'BASE TABLE',
        columns: [column('CustomerName')],
      },
      {
        schema: 'dbo',
        name: 'vw_OpenOrders',
        kind: 'VIEW',
        columns: [column('OrderId'), column('OpenAmount')],
      },
    ],
    'c1:Archive': [
      {
        schema: 'dbo',
        name: 'OrdersHistory',
        kind: 'BASE TABLE',
        columns: [column('ArchivedOrderId')],
      },
    ],
    'c2:Warehouse': [
      {
        schema: 'fact',
        name: 'Sales',
        kind: 'BASE TABLE',
        columns: [column('Quantity')],
      },
    ],
  };

  test('an empty query leaves the tree untouched', () => {
    const index = buildExplorerFilterIndex({
      query: '   ',
      connections,
      tablesByDb,
      routinesByDb,
      schemasByDb,
    });

    assert.equal(index.active, false);
    assert.equal(index.databases.size, 0);
    assert.equal(index.tables.size, 0);
    assert.equal(index.procedures.size, 0);
    assert.equal(index.functions.size, 0);
  });

  test('a column hit keeps the owning table and records only the matching columns', () => {
    const index = buildExplorerFilterIndex({
      query: 'customerid',
      connections,
      tablesByDb,
      routinesByDb,
      schemasByDb,
    });

    assert.equal(index.active, true);
    assert.ok(index.databases.has(explorerDatabaseKey('c1', 'AppDb')));
    assert.equal(index.databases.has(explorerDatabaseKey('c1', 'Archive')), false);

    const match = index.tables.get(explorerTableKey('c1', 'AppDb', 'dbo', 'Orders'));
    assert.ok(match);
    assert.equal(match?.allColumns, false);
    assert.deepEqual([...(match?.columns ?? [])], ['customerid']);

    // The schema snapshot is carried in the index so rendering does not re-scan the store.
    assert.equal(
      index.schemaColumns.get(explorerTableKey('c1', 'AppDb', 'dbo', 'Orders'))?.length,
      3
    );

    // The parent nodes are revealed so the match is visible without another click.
    assert.ok(index.expandDatabases.has(explorerDatabaseKey('c1', 'AppDb')));
    assert.ok(index.expandConnections.has('c1'));
  });

  test('a database name hit keeps every loaded table under that database', () => {
    const index = buildExplorerFilterIndex({
      query: 'archive',
      connections,
      tablesByDb,
      routinesByDb,
      schemasByDb,
    });

    assert.ok(index.databases.has(explorerDatabaseKey('c1', 'Archive')));
    assert.equal(index.databases.has(explorerDatabaseKey('c1', 'AppDb')), false);

    const match = index.tables.get(explorerTableKey('c1', 'Archive', 'dbo', 'OrdersHistory'));
    assert.equal(match?.allColumns, true);
  });

  test('a table name hit keeps all of its columns', () => {
    const index = buildExplorerFilterIndex({
      query: 'vw_open',
      connections,
      tablesByDb,
      routinesByDb,
      schemasByDb,
    });

    const match = index.tables.get(
      explorerTableKey('c1', 'AppDb', 'dbo', 'vw_OpenOrders')
    );
    assert.equal(match?.allColumns, true);
    assert.equal(match?.columns.size, 0, 'a name hit does not need column-level filtering');
  });

  test('procedures, functions and cross-connection hits are indexed', () => {
    const procedureIndex = buildExplorerFilterIndex({
      query: 'report',
      connections,
      tablesByDb,
      routinesByDb,
      schemasByDb,
    });
    assert.ok(
      procedureIndex.procedures.has(explorerRoutineKey('c1', 'AppDb', 'dbo', 'usp_Report'))
    );

    const crossConnection = buildExplorerFilterIndex({
      query: 'quantity',
      connections,
      tablesByDb,
      routinesByDb,
      schemasByDb,
    });
    assert.ok(crossConnection.databases.has(explorerDatabaseKey('c2', 'Warehouse')));
    assert.ok(crossConnection.expandConnections.has('c2'));
  });

  test('hidden tables are excluded from object and column matches', () => {
    const index = buildExplorerFilterIndex({
      query: 'customerid',
      connections,
      tablesByDb,
      routinesByDb,
      schemasByDb,
      // Views deliberately do not obey the hidden-table rules, matching the Explorer's lists.
      isTableHidden: (name) => name === 'Orders',
    });

    assert.equal(index.tables.size, 0);
    assert.equal(index.databases.size, 0);
  });

  test('matchingColumns is a case-insensitive substring match', () => {
    const columns = [column('CustomerId'), column('CompanyName')];
    assert.deepEqual(
      matchingColumns(columns, 'CUSTOMER').map((entry) => entry.name),
      ['CustomerId']
    );
    assert.deepEqual(matchingColumns(columns, 'name').map((entry) => entry.name), [
      'CompanyName',
    ]);
    assert.equal(matchingColumns(columns, '').length, 0);
    assert.equal(matchingColumns(undefined, 'x').length, 0);
  });
});
