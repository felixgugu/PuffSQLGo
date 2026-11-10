import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import './assets/main.css';
import 'tabulator-tables/dist/css/tabulator.css';
import './styles/tabulatorTheme.css';
import './styles/editorGuard.css';

import tooltip from './directives/tooltip';

import { i18n } from './i18n';
import { errorLogService } from './services/errorLogService';

// Prevent default browser/system context menu globally for a native desktop feel
window.addEventListener('contextmenu', (e) => {
  e.preventDefault();
});

const app = createApp(App);
const pinia = createPinia();

function describeError(value: unknown): { message: string; stack?: string } {
  if (value instanceof Error) {
    return { message: value.message || value.name, stack: value.stack };
  }
  if (typeof value === 'string') {
    return { message: value };
  }
  try {
    return { message: JSON.stringify(value) };
  } catch {
    return { message: String(value) };
  }
}

// Mirror uncaught UI failures into the same file as backend errors. These handlers are silent
// when the app runs in a browser without the Wails bridge.
const recentClientErrors = new Map<string, number>();

function logClientError(message: string, stack?: string): void {
  const now = Date.now();
  if (now - (recentClientErrors.get(message) ?? 0) < 5000) {
    return;
  }
  if (recentClientErrors.size > 100) {
    recentClientErrors.clear();
  }
  recentClientErrors.set(message, now);
  void errorLogService.log('ERROR', message, stack);
}

app.config.errorHandler = (err, _instance, info) => {
  const detail = describeError(err);
  logClientError(`vue:${info}: ${detail.message}`, detail.stack);
  console.error(err);
};

window.addEventListener('error', (event) => {
  // Resource load failures surface here without an Error object; they are not actionable.
  if (!event.error && !event.message) return;
  const detail = describeError(event.error ?? event.message);
  logClientError(`window.error: ${detail.message}`, detail.stack);
});

window.addEventListener('unhandledrejection', (event) => {
  const detail = describeError(event.reason);
  logClientError(`unhandledrejection: ${detail.message}`, detail.stack);
});

// Read initial saved locale before the first render so the app never flashes the wrong language.
let initialLocale = 'zh-TW';
try {
  const savedSettings = localStorage.getItem('sqlight_app_settings');
  if (savedSettings) {
    const parsed = JSON.parse(savedSettings);
    if (parsed.locale === 'en' || parsed.locale === 'zh-TW') {
      initialLocale = parsed.locale;
    }
  }
} catch {
  // fallback to default
}

(i18n.global.locale as any).value = initialLocale;

app.use(pinia);
app.use(i18n);
app.directive('tooltip', tooltip);

app.mount('#app');
