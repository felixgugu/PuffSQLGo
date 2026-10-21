import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
import { describe, test } from 'node:test';
import {
  projectPath,
  readProjectSource,
  readSource,
  trackedSources,
} from './helpers/source';

describe('application metadata', () => {
  test('package, lockfile and Wails installer versions agree', () => {
    const pkg = JSON.parse(readSource('package.json')) as { version: string };
    const lock = JSON.parse(readSource('package-lock.json')) as {
      version: string;
      packages: Record<string, { version?: string }>;
    };
    const wails = JSON.parse(readProjectSource('wails.json')) as {
      outputfilename: string;
      info: { productName: string; productVersion: string };
    };

    assert.match(pkg.version, /^\d+\.\d+\.\d+$/);
    assert.equal(lock.version, pkg.version);
    assert.equal(lock.packages['']?.version, pkg.version);
    assert.equal(wails.info.productVersion, pkg.version);
    assert.equal(wails.outputfilename, 'PuffSQL');
    assert.equal(wails.info.productName, 'PuffSQL');
  });

  test('Vite is the only source of the version shown by the UI', () => {
    const config = readSource('vite.config.ts');
    assert.match(config, /__APP_VERSION__: JSON\.stringify\(appVersion\)/);
    assert.match(readSource('src/vite-env.d.ts'), /declare const __APP_VERSION__: string;/);
    for (const file of [
      'src/components/layout/AppStatusBar.vue',
      'src/components/modals/SettingsModal.vue',
    ]) {
      const source = readSource(file);
      assert.match(source, /__APP_VERSION__/, `${file} must use the injected version`);
      assert.doesNotMatch(source, /v\d+\.\d+\.\d+/, `${file} must not hard-code a version`);
    }
  });

  test('the Go module and Wails entry point use the PuffSQL executable name', () => {
    const module = readProjectSource('go.mod');
    const main = readProjectSource('main.go');
    assert.match(module, /^module puffsql$/m);
    assert.match(main, /wails\.Run\(&options\.App\{/);
    assert.match(main, /Bind:\s+\[\]any\{app\}/);
    assert.match(main, /Title:\s+"PuffSQL - Lightweight SQL Client"/);
  });

  test('portable packaging emits PuffSQL.exe and PuffSQL-Portable.zip', () => {
    const script = readProjectSource('build-portable.ps1');
    assert.match(script, /build\\bin\\PuffSQL\.exe/);
    assert.match(script, /PuffSQL-Portable\.zip/);
    assert.match(script, /sql_custom_templates\.json/);
    assert.match(readProjectSource('build-portable.bat'), /PuffSQL - Portable Build Launcher/);
  });
});

describe('Go backend contracts', () => {
  test('application data, logs and OS keychain use PuffSQL names', () => {
    assert.match(readProjectSource('internal/storage/storage.go'), /appDirName\s+= "PuffSQL"/);
    assert.match(readProjectSource('internal/storage/storage.go'), /"puffsql\.log"/);
    assert.match(readProjectSource('internal/storage/storage.go'), /"ai\.log"/);
    assert.match(
      readProjectSource('internal/connection/credentials.go'),
      /credentialService\s+= "PuffSQLDesktopClient"/
    );
  });

  test('legacy profile and keychain data are migrated', () => {
    const storage = readProjectSource('internal/storage/storage.go');
    const credentials = readProjectSource('internal/connection/credentials.go');
    assert.match(storage, /legacyDirName\s+= "SQLight"/);
    assert.match(storage, /os\.Rename\(legacy, current\)/);
    assert.match(credentials, /legacyCredentialService\s+= "SQLightDesktopClient"/);
    assert.match(credentials, /keyring\.Set\(credentialService, id, password\)/);
  });

  test('all frontend commands have a Wails bridge and a browser fallback', () => {
    const api = readSource('src/services/api.ts');
    const commands = [
      'get_connections',
      'save_connection',
      'execute_query',
      'cancel_query',
      'get_databases',
      'get_table_objects',
      'get_table_import_capabilities',
      'import_table_rows',
      'pick_sql_folder',
      'load_custom_templates',
      'log_ai_request',
    ];
    for (const command of commands) {
      assert.ok(api.includes(`case '${command}':`), `${command} must be dispatched`);
    }
    assert.match(api, /export const isWails = \(\): boolean/);
    assert.doesNotMatch(api, /@tauri-apps/);
  });

  test('generated Wails bindings cover every backend command category', () => {
    const bindings = readProjectSource('frontend/wailsjs/go/main/App.d.ts');
    for (const method of [
      'CancelQuery',
      'ClearAILog',
      'Connect',
      'DeleteConnection',
      'Disconnect',
      'ExecuteQuery',
      'ExportDatabaseSchemaToExcel',
      'ExportDatabaseTablesToExcel',
      'GetAILogPath',
      'GetColumns',
      'GetConnections',
      'GetConnectionSPID',
      'GetDatabases',
      'GetDatabaseSchema',
      'GetForeignKeys',
      'GetQueryLogPath',
      'GetTableImportCapabilities',
      'GetTableObjects',
      'GetTables',
      'ImportTableRows',
      'LoadCustomTemplates',
      'LogAIRequest',
      'LogAIResponse',
      'OpenAILogFile',
      'OpenCustomTemplatesFile',
      'OpenQueryLogFile',
      'PickSQLFolder',
      'PreviewColumnChange',
      'ReadSQLFile',
      'RenameSQLPath',
      'SaveConnection',
      'SaveCustomTemplates',
      'ScanSQLFolder',
      'SwitchDatabase',
      'TestConnection',
      'WriteSQLFile',
      'ApplyColumnChange',
    ]) {
      assert.match(bindings, new RegExp(`export function ${method}\\(`));
    }
  });
});

describe('desktop shell', () => {
  test('Wails creates a frameless resizable main window', () => {
    const main = readProjectSource('main.go');
    assert.match(main, /frameless := runtime\.GOOS == "windows"/);
    assert.match(main, /Frameless:\s+frameless/);
    assert.match(main, /MinWidth:\s+1368/);
    assert.match(main, /MinHeight:\s+600/);
    assert.match(main, /DisableResize:\s+false/);
  });

  test('AppHeader is the Wails drag region and hosts window controls', () => {
    const source = readSource('src/components/layout/AppHeader.vue');
    assert.match(source, /--wails-draggable:\s*drag/);
    assert.doesNotMatch(source, /data-tauri-drag-region/);
    assert.match(source, /class="h-10 bg-dark-850 border-b border-dark-700/);
    assert.match(source, /const showWindowControls = windowService\.isCustomTitleBar\(\)/);
    assert.match(source, /<WindowControls v-if="showWindowControls" \/>/);
  });

  test('title bar carries controls without a decorative logo cell', () => {
    const source = readSource('src/components/layout/AppHeader.vue');
    assert.doesNotMatch(source, /appIcon/);
    assert.doesNotMatch(source, /<img\b/);
    assert.doesNotMatch(source, /min-w-\[40px\]/);
  });

  test('status bar shows the product and version without a decorative mark', () => {
    const source = readSource('src/components/layout/AppStatusBar.vue');
    assert.match(source, /PuffSQL v\{\{ appVersion \}\}/);
    assert.doesNotMatch(source, /🐡|⭐|🚀/);
    assert.doesNotMatch(source, /<img\b/);
  });

  test('About reuses the shipped PuffSQL logo', () => {
    const source = readSource('src/components/modals/SettingsModal.vue');
    assert.match(source, /import appIcon from '\.\.\/\.\.\/assets\/puffsql_logo\.png'/);
    assert.match(source, /<img[\s\S]*?:src="appIcon"[\s\S]*?alt="PuffSQL"/);
    assert.ok(existsSync(projectPath('frontend/src/assets/puffsql_logo.png')));
    assert.ok(existsSync(projectPath('build/appicon.png')));
  });

  test('WindowControls wires Wails minimize, maximize and close actions', () => {
    const source = readSource('src/components/layout/WindowControls.vue');
    for (const marker of [
      'windowService.minimize()',
      'windowService.toggleMaximize()',
      'windowService.close()',
      'windowService.isMaximized()',
      'windowService.onResized(',
    ]) {
      assert.ok(source.includes(marker), `${marker} should be wired`);
    }
    assert.match(source, /aria-label="最小化"/);
    assert.match(source, /aria-label="關閉"/);
  });

  test('windowService delegates to generated Wails runtime with a browser fallback', () => {
    const source = readSource('src/services/windowService.ts');
    assert.match(source, /import \{ isWails \} from '\.\/api'/);
    assert.match(source, /from '\.\.\/\.\.\/wailsjs\/runtime\/runtime'/);
    assert.match(source, /if \(!isWails\(\)\) return false/);
    assert.match(source, /WindowIsMaximised\(\)/);
    assert.match(source, /WindowMinimise\(\)/);
    assert.match(source, /WindowToggleMaximise\(\)/);
    assert.match(source, /Quit\(\)/);
    assert.match(source, /window\.addEventListener\('resize', listener\)/);
  });

  test('no user-facing Tauri dependency remains in the frontend sources', () => {
    for (const file of trackedSources()) {
      const source = readSource(file);
      assert.doesNotMatch(source, /@tauri-apps/, `${file} still imports Tauri`);
      assert.doesNotMatch(source, /data-tauri-drag-region/, `${file} still uses Tauri drag markup`);
    }
  });

  test('no renderer uses an emoji as structural chrome', () => {
    for (const file of [
      'src/components/layout/AppHeader.vue',
      'src/components/layout/AppStatusBar.vue',
    ]) {
      const source = readSource(file);
      const template = source.slice(0, source.indexOf('</template>'));
      assert.doesNotMatch(template, /[\u{1F300}-\u{1FAFF}]/u, `${file} uses vector icons`);
    }
  });

  test('context-menu suppression and application mounting stay wired', () => {
    assert.match(readSource('index.html'), /contextmenu/);
    assert.match(readSource('src/main.ts'), /contextmenu/);
    assert.match(readSource('index.html'), /id="app"/);
    assert.ok(existsSync(projectPath('frontend/src/App.vue')));
  });
});
