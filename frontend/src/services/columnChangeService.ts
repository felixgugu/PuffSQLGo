import { invokeCommand } from './api';
import type {
  ColumnChangePreview,
  ColumnChangeRequest,
  ColumnChangeResult,
} from '@/types/columnChange';

export const columnChangeService = {
  preview(req: ColumnChangeRequest): Promise<ColumnChangePreview> {
    return invokeCommand<ColumnChangePreview>('preview_column_change', { req });
  },

  apply(req: ColumnChangeRequest, fingerprint: string): Promise<ColumnChangeResult> {
    return invokeCommand<ColumnChangeResult>('apply_column_change', { req, fingerprint });
  },
};
