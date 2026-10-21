export type ColumnChangeOperation = 'add' | 'alter' | 'drop';

export interface ColumnChangeRequest {
  connectionId: string;
  database: string;
  schema: string;
  table: string;
  operation: ColumnChangeOperation;
  originalName?: string;
  columnName: string;
  dataType: string;
  length?: number | null;
  precision?: number | null;
  scale?: number | null;
  isNullable: boolean;
  defaultExpression?: string | null;
  description?: string | null;
}

export interface ColumnDependency {
  kind: string;
  name: string;
  detail: string;
}

export interface ColumnChangeIssue {
  code: string;
  message: string;
}

export interface ColumnChangePreview {
  sql: string;
  fingerprint: string;
  warnings: ColumnChangeIssue[];
  blockingIssues: ColumnChangeIssue[];
  dependencies: ColumnDependency[];
  affectedRows: number;
}

export interface ColumnChangeResult {
  executionTimeMs: number;
}
