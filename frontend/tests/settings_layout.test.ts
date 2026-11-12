import assert from 'node:assert/strict';
import { before, beforeEach, describe, test } from 'node:test';
import { createPinia, setActivePinia } from 'pinia';
import { useSettingsStore } from '../src/stores/settingsStore';
import { readSource } from './helpers/source';

const storage = new Map<string, string>();

before(() => {
  (globalThis as any).localStorage = {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, String(value));
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => storage.clear(),
    key: () => null,
    length: 0,
  };
});

beforeEach(() => {
  storage.clear();
  setActivePinia(createPinia());
});

describe('settings_row_layout', () => {
  test('the label column is fixed and the description spans both columns', () => {
    const row = readSource('src/components/modals/settings/SettingsRow.vue');

    assert.match(row, /grid-template-columns:\s*160px minmax\(0, 1fr\)/);
    assert.match(row, /grid-column:\s*1 \/ -1/);
    assert.match(row, /class="settings-row__label font-medium text-dark-100"/);
    assert.match(row, /class="settings-row__desc text-xxs text-dark-500"/);
    assert.match(row, /align-items:\s*start/);
    assert.match(row, /justify-content:\s*flex-start/);
    assert.match(row, /padding:\s*12px 0/);
  });

  test('sections group rows and draw one hairline between them', () => {
    const section = readSource('src/components/modals/settings/SettingsSection.vue');

    assert.match(section, /divide-y divide-dark-750\/70/);
    assert.match(section, /settings-section__title/);
    assert.match(section, /settings-section__desc/);
  });

  test('the standard settings tabs use the shared row and section components', () => {
    const modal = readSource('src/components/modals/SettingsModal.vue');

    assert.match(modal, /import SettingsRow from '\.\/settings\/SettingsRow\.vue'/);
    assert.match(modal, /import SettingsSection from '\.\/settings\/SettingsSection\.vue'/);
    const rows = modal.match(/<SettingsRow/g) ?? [];
    const sections = modal.match(/<SettingsSection/g) ?? [];
    assert.ok(rows.length >= 12, `expected shared rows across settings, found ${rows.length}`);
    assert.ok(sections.length >= 10, `expected grouped sections, found ${sections.length}`);
    assert.match(modal, /settingsStore\.showCommitSqlPreview/);
    assert.match(modal, /:label="\$t\('settingsModal\.showCommitSqlPreview'\)"/);
  });

  test('the sidebar owns section navigation and the content pane shows its description', () => {
    const modal = readSource('src/components/modals/SettingsModal.vue');

    assert.match(modal, /class="settings-nav /);
    assert.match(modal, /activeTabMeta\.label/);
    assert.match(modal, /activeTabMeta\.desc/);
    assert.match(modal, /settingsModal\.pageDesc\.appearance/);
    assert.match(modal, /:aria-current="activeTab === tab\.id \? 'page' : undefined"/);
  });
});

describe('commit_sql_preview_setting', () => {
  test('defaults to off, persists and resets', () => {
    const store = useSettingsStore();

    assert.equal(store.showCommitSqlPreview, false, 'the preview is opt-in');
    store.showCommitSqlPreview = true;
    store.saveSettings();

    const saved = JSON.parse(storage.get('sqlight_app_settings') || '{}');
    assert.equal(saved.showCommitSqlPreview, true);

    store.resetToDefaults();
    assert.equal(store.showCommitSqlPreview, false);
  });

  test('the result grid executes directly unless the preview or connection prompt asks for it', () => {
    const grid = readSource('src/components/results/ResultGridItem.vue');

    assert.match(
      grid,
      /settingsStore\.showCommitSqlPreview \|\| requiresModificationPrompt\.value/
    );
    assert.match(grid, /await executeCommit\(true\)/);
    assert.match(grid, /@click="handleCommitAction"/);
    assert.match(grid, /function reportCommitError/);
  });
});
