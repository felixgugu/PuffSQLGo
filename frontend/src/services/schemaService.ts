import { invokeCommand } from './api';
import type {
  DatabaseItem,
  TableItem,
  ColumnItem,
  TableObjects,
  TableSchema,
  ForeignKeyItem,
} from '@/types/schema';

export const TABLE_OBJECTS_TIMEOUT_MS = 25_000;

async function withTimeout<T>(promise: Promise<T>, milliseconds: number, message: string): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(new Error(message)), milliseconds);
  });
  try {
    return await Promise.race([promise, timeout]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

export const schemaService = {
  async getDatabases(connectionId: string): Promise<DatabaseItem[]> {
    return invokeCommand<DatabaseItem[]>('get_databases', { connectionId });
  },

  async getTables(
    connectionId: string,
    database?: string,
    schema?: string
  ): Promise<TableItem[]> {
    return invokeCommand<TableItem[]>('get_tables', {
      connectionId,
      database,
      schema,
    });
  },

  async getColumns(
    connectionId: string,
    schema: string,
    table: string,
    database?: string
  ): Promise<ColumnItem[]> {
    return invokeCommand<ColumnItem[]>('get_columns', {
      connectionId,
      schema,
      table,
      database,
    });
  },

  async getForeignKeys(
    connectionId: string,
    database?: string,
    schema?: string,
    table?: string
  ): Promise<ForeignKeyItem[]> {
    return invokeCommand<ForeignKeyItem[]>('get_foreign_keys', {
      connectionId,
      database,
      schema,
      table,
    });
  },

  /**
   * Columns, constraints, indexes and triggers of one object, in a single round trip — what the
   * Explorer lists under a table or view node, and what the CREATE script is built from.
   */
  async getTableObjects(
    connectionId: string,
    schema: string,
    table: string,
    database?: string
  ): Promise<TableObjects> {
    return withTimeout(
      invokeCommand<TableObjects>('get_table_objects', {
        connectionId,
        schema,
        table,
        database,
      }),
      TABLE_OBJECTS_TIMEOUT_MS,
      `載入資料表 ${schema}.${table} 逾時，請重試`
    );
  },

  async getDatabaseSchema(
    connectionId: string,
    database?: string
  ): Promise<TableSchema[]> {
    return invokeCommand<TableSchema[]>('get_database_schema', {
      connectionId,
      database,
    });
  },

  async switchDatabase(connectionId: string, database: string): Promise<void> {
    return invokeCommand<void>('switch_database', { connectionId, database });
  },
};
