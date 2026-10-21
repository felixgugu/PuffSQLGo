import * as monaco from 'monaco-editor';
import EditorWorker from './editorWorker?worker';
import { buildEditorTheme } from './editorThemeTokens';
import { DEFAULT_SURFACE } from '@/services/themeManager';

self.MonacoEnvironment = {
  getWorker: function (_moduleId: unknown, _label: string) {
    return new EditorWorker();
  },
};

function persistedKeywordBold(): boolean {
  try {
    const raw = localStorage.getItem('sqlight_app_settings');
    return raw ? JSON.parse(raw).editorKeywordsBold !== false : true;
  } catch {
    return true;
  }
}

export function ensureSqlightTheme(surfaceName = DEFAULT_SURFACE, keywordsBold = true) {
  const dark = buildEditorTheme(surfaceName, 'dark', keywordsBold);
  const light = buildEditorTheme(surfaceName, 'light', keywordsBold);

  monaco.editor.defineTheme('sqlight-dark', { inherit: true, ...dark });
  monaco.editor.defineTheme('sqlight-light', { inherit: true, ...light });
}

// Automatically register theme on module load
ensureSqlightTheme(DEFAULT_SURFACE, persistedKeywordBold());

// Listen for surface changes and update Monaco theme dynamically
if (typeof window !== 'undefined') {
  window.addEventListener('sqlight:surface-changed', (e: any) => {
    const surfaceName = e.detail?.surface || DEFAULT_SURFACE;
    ensureSqlightTheme(surfaceName, persistedKeywordBold());
    const isDark = document.documentElement.classList.contains('dark');
    monaco.editor.setTheme(isDark ? 'sqlight-dark' : 'sqlight-light');
  });
}

export { monaco };
