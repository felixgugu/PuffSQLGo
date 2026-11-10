import type { ColumnItem, RoutineItem, TableItem, TableSchema } from '@/types/schema';

/**
 * Pure filtering model behind the Explorer's search box.
 *
 * The tree is filtered in one pass instead of walking every database from the template: a single
 * index answers "is this connection/database/table included?" and records which columns matched,
 * so the tree can auto-expand the parent nodes that contain a hit. Keeping this free of Pinia and
 * DOM access makes the matching rules unit-testable.
 */

export interface ExplorerConnectionRef {
  id: string;
  name: string;
  databases: string[];
}

export interface ExplorerFilterSources {
  query: string;
  connections: ExplorerConnectionRef[];
  tablesByDb: Record<string, TableItem[] | undefined>;
  routinesByDb: Record<string, RoutineItem[] | undefined>;
  schemasByDb: Record<string, TableSchema[] | undefined>;
  isDatabaseHidden?: (name: string) => boolean;
  isTableHidden?: (name: string, schema?: string) => boolean;
}

export interface ExplorerTableMatch {
  /** The table/view (or an ancestor) matched, so every column should stay visible. */
  allColumns: boolean;
  /** Lower-cased names of the columns that matched the query. */
  columns: Set<string>;
}

export interface ExplorerFilterIndex {
  active: boolean;
  /** `connId:database` keys of the databases that should stay in the tree. */
  databases: Set<string>;
  /** Connections whose matching databases should be revealed automatically. */
  expandConnections: Set<string>;
  /** Databases whose matching objects should be revealed automatically. */
  expandDatabases: Set<string>;
  /** `connId:database:schema.table` → what to show under that node. */
  tables: Map<string, ExplorerTableMatch>;
  /** Columns loaded from the database schema for each included table (empty when unavailable). */
  schemaColumns: Map<string, ColumnItem[]>;
  /** `connId:database:schema.name` keys of the matching routines. */
  procedures: Set<string>;
  functions: Set<string>;
}

export function explorerDatabaseKey(connId: string, database: string): string {
  return `${connId}:${database}`;
}

export function explorerTableKey(
  connId: string,
  database: string,
  schema: string,
  table: string
): string {
  return `${connId}:${database}:${schema}.${table}`;
}

export function explorerRoutineKey(
  connId: string,
  database: string,
  schema: string,
  name: string
): string {
  return `${connId}:${database}:${schema}.${name}`;
}

/** Columns whose name contains the query (case-insensitive substring). */
export function matchingColumns(
  columns: ColumnItem[] | undefined,
  query: string
): ColumnItem[] {
  const needle = query.trim().toLowerCase();
  if (!needle || !columns || columns.length === 0) return [];
  return columns.filter((column) => column.name.toLowerCase().includes(needle));
}

function emptyIndex(active = false): ExplorerFilterIndex {
  return {
    active,
    databases: new Set<string>(),
    expandConnections: new Set<string>(),
    expandDatabases: new Set<string>(),
    tables: new Map<string, ExplorerTableMatch>(),
    schemaColumns: new Map<string, ColumnItem[]>(),
    procedures: new Set<string>(),
    functions: new Set<string>(),
  };
}

export function buildExplorerFilterIndex(
  sources: ExplorerFilterSources
): ExplorerFilterIndex {
  const needle = sources.query.trim().toLowerCase();
  if (!needle) return emptyIndex(false);

  const index = emptyIndex(true);

  for (const connection of sources.connections) {
    const connectionNameHit = connection.name.toLowerCase().includes(needle);
    const databases = connection.databases.filter(
      (database) => !sources.isDatabaseHidden?.(database)
    );

    for (const database of databases) {
      const dbKey = explorerDatabaseKey(connection.id, database);
      const databaseNameHit = connectionNameHit || database.toLowerCase().includes(needle);
      let databaseIncluded = databaseNameHit;
      let databaseHasContent = false;

      const schemaByName = new Map<string, TableSchema>();
      for (const table of sources.schemasByDb[dbKey] ?? []) {
        schemaByName.set(`${table.schema}.${table.name}`, table);
      }

      for (const table of sources.tablesByDb[dbKey] ?? []) {
        // Hidden-object rules only ever applied to base tables; views keep their current behaviour.
        if (
          table.kind === 'BASE TABLE' &&
          sources.isTableHidden?.(table.name, table.schema)
        ) {
          continue;
        }

        const tableNameHit =
          databaseNameHit ||
          table.name.toLowerCase().includes(needle) ||
          table.schema.toLowerCase().includes(needle);
        // Compute the column hits even when the table name matched: they decide whether the node
        // should also auto-expand, and they keep the match visible when the table name is shared.
        const tableKey = explorerTableKey(connection.id, database, table.schema, table.name);
        const schemaColumns = schemaByName.get(`${table.schema}.${table.name}`)?.columns;
        const matchedColumns = matchingColumns(schemaColumns, needle);

        if (!tableNameHit && matchedColumns.length === 0) continue;

        index.tables.set(tableKey, {
          allColumns: tableNameHit,
          columns: new Set(matchedColumns.map((column) => column.name.toLowerCase())),
        });
        // Cache the column list in the index so the tree does not re-scan every schema on render.
        index.schemaColumns.set(tableKey, schemaColumns ?? []);
        databaseIncluded = true;
        databaseHasContent = true;
      }

      for (const routine of sources.routinesByDb[dbKey] ?? []) {
        const routineHit =
          databaseNameHit ||
          routine.name.toLowerCase().includes(needle) ||
          routine.schema.toLowerCase().includes(needle);
        if (!routineHit) continue;

        const key = explorerRoutineKey(
          connection.id,
          database,
          routine.schema,
          routine.name
        );
        if (routine.kind === 'PROCEDURE') {
          index.procedures.add(key);
        } else {
          index.functions.add(key);
        }
        databaseIncluded = true;
        databaseHasContent = true;
      }

      if (databaseIncluded) {
        index.databases.add(dbKey);
      }
      if (databaseHasContent) {
        index.expandDatabases.add(dbKey);
        index.expandConnections.add(connection.id);
      }
      if (connectionNameHit) {
        index.expandConnections.add(connection.id);
      }
    }
  }

  return index;
}
