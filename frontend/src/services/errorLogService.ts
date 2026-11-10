import { invokeCommand } from './api';

/**
 * Writes diagnostics to the PuffSQL-error.log file that sits beside the executable. Client-side
 * failures are mirrored there so a crash or a rejected promise can be matched with the backend
 * trace for the same session.
 */
export const errorLogService = {
  async getPath(): Promise<string> {
    return invokeCommand<string>('get_error_log_path');
  },

  async open(): Promise<string> {
    return invokeCommand<string>('open_error_log');
  },

  async log(level: string, message: string, stack?: string): Promise<void> {
    try {
      await invokeCommand<void>('log_client_error', {
        level,
        message,
        stack: stack ?? '',
      });
    } catch {
      // Logging must never become the reason the UI fails.
    }
  },
};
