<template>
  <SqDialog
    :visible="isOpen"
    dismissable-mask
    :class="dialogWindow.marker"
    class="w-[860px] h-[80vh] max-w-[95vw] max-h-[92vh]"
    content-class="!p-0 flex flex-col overflow-hidden h-full"
    @show="dialogWindow.onShow"
    @hide="dialogWindow.onHide"
    @update:visible="val => !val && $emit('close')"
  >
    <template #header>
      <Settings class="w-4 h-4 text-accent" />
      <span>{{ $t('settingsModal.title') }}</span>
    </template>

    <!-- Settings navigation + content -->
    <div class="flex flex-1 min-h-0 max-[720px]:flex-col">
      <nav
        class="settings-nav w-[200px] shrink-0 overflow-y-auto border-r border-dark-750/70 bg-dark-900/40 p-2 space-y-1 max-[720px]:flex max-[720px]:w-full max-[720px]:gap-1 max-[720px]:space-y-0 max-[720px]:overflow-x-auto max-[720px]:overflow-y-hidden max-[720px]:border-b max-[720px]:border-r-0"
      >
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="relative flex w-full shrink-0 items-center gap-2.5 rounded-md px-3 py-2 text-left text-xs transition-colors max-[720px]:w-auto"
          :class="
            activeTab === tab.id
              ? 'bg-dark-750/60 text-dark-100 font-medium'
              : 'text-dark-400 hover:bg-dark-750/40 hover:text-dark-200'
          "
          :aria-current="activeTab === tab.id ? 'page' : undefined"
          @click="activeTab = tab.id"
        >
          <span
            v-if="activeTab === tab.id"
            class="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-full bg-accent max-[720px]:hidden"
          />
          <component
            :is="tab.icon"
            class="w-3.5 h-3.5 shrink-0"
            :class="activeTab === tab.id ? 'text-accent' : 'text-dark-500'"
          />
          <span class="truncate">{{ tab.label }}</span>
        </button>
      </nav>

      <div class="flex-1 min-w-0 overflow-y-auto text-xs text-dark-200">
        <div class="mx-auto w-full max-w-[720px] space-y-7 px-7 py-6">
          <header class="space-y-1">
            <h2 class="text-base font-semibold text-dark-100">{{ activeTabMeta.label }}</h2>
            <p class="text-xs text-dark-500">{{ activeTabMeta.desc }}</p>
          </header>
          <!-- Tab 0: Appearance & Theme Settings -->
          <div v-if="activeTab === 'theme'" class="space-y-7">
            <SettingsSection
              :title="$t('settingsModal.sections.interface')"
              :description="$t('settingsModal.sections.interfaceDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.language')"
                :description="$t('settingsModal.languageDesc')"
              >
                <SqSelect
                  :model-value="settingsStore.locale"
                  :options="SUPPORTED_LOCALES"
                  option-label="label"
                  option-value="value"
                  class="w-full max-w-[240px]"
                  @update:model-value="handleLocaleChange"
                />
              </SettingsRow>

              <SettingsRow
                :label="$t('settingsModal.globalFont')"
                :description="$t('settingsModal.globalFontDesc')"
              >
                <div class="w-full max-w-[320px] space-y-1">
                  <FontFamilyPicker
                    :model-value="settingsStore.globalFontFamily"
                    :options="globalFontSelectOptions"
                    :default-value="DEFAULT_GLOBAL_FONT_FAMILY"
                    select-class="w-full !text-xs !bg-dark-900 !border-dark-700"
                    preview
                    @update:model-value="handleGlobalFontChange"
                  />
                  <span v-if="selectedGlobalFont" class="text-xxs text-dark-500 block">
                    {{ selectedGlobalFont.description }}
                  </span>
                </div>
              </SettingsRow>

              <SettingsRow :label="$t('settingsModal.uiFontSize')">
                <template #description>
                  {{
                    $t('settingsModal.uiFontSizeDesc', {
                      primary: settingsStore.uiFontSize,
                      secondary: settingsStore.uiFontSize - 1,
                      heading: settingsStore.uiFontSize + 2,
                    })
                  }}
                </template>
                <div class="w-full max-w-[420px] space-y-2">
                  <SqSelect
                    :model-value="settingsStore.uiFontSize"
                    :options="uiFontSizeOptions"
                    option-label="label"
                    option-value="value"
                    class="w-full max-w-[140px]"
                    @update:model-value="handleUiFontSizeChange"
                  />
                  <div class="rounded-md border border-dark-750 bg-dark-900/70 px-3 py-2 space-y-0.5">
                    <div class="text-base font-semibold text-dark-100">
                      {{ $t('settingsModal.uiFontSizePreviewHeading') }}
                    </div>
                    <div class="text-sm text-dark-200">{{ $t('settingsModal.uiFontSizePreviewPrimary') }}</div>
                    <div class="text-xs text-dark-400">{{ $t('settingsModal.uiFontSizePreviewSecondary') }}</div>
                  </div>
                </div>
              </SettingsRow>
            </SettingsSection>

            <SettingsSection
              :title="$t('settingsModal.sections.color')"
              :description="$t('settingsModal.sections.colorDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.colorMode')"
                :description="$t('settingsModal.colorModeDesc')"
              >
                <SqSegmented
                  :model-value="settingsStore.colorMode"
                  :options="[
                    { label: $t('settingsModal.darkMode'), value: 'dark' },
                    { label: $t('settingsModal.lightMode'), value: 'light' },
                  ]"
                  @update:model-value="val => settingsStore.setColorMode(val as 'dark' | 'light')"
                />
              </SettingsRow>
            </SettingsSection>

            <SettingsSection
              :title="$t('settingsModal.sections.preview')"
              :description="$t('settingsModal.sections.previewDesc')"
            >
              <div class="py-3">
                <div class="flex flex-wrap items-center gap-2.5">
                  <SqButton label="主要按鈕" tone="accent" variant="solid" size="sm">
                    <template #icon><Check class="w-3.5 h-3.5" /></template>
                  </SqButton>
                  <SqButton label="次要外框" variant="outline" size="sm" />
                  <SqButton label="文字按鈕" size="sm" />
                  <SqChip label="狀態標籤" />
                  <SqChip tone="accent" label="12" />
                  <SqInput placeholder="輸入框預覽..." size="sm" class="w-32" />
                  <div class="flex items-center space-x-1.5 pl-2">
                    <SqCheckbox :model-value="true" />
                    <span class="text-xxs text-dark-300">核取方塊</span>
                  </div>
                </div>
              </div>
            </SettingsSection>
          </div>

          <!-- Tab 1: Editor Settings -->
          <div v-else-if="activeTab === 'editor'" class="space-y-7">
            <SettingsSection
              :title="$t('settingsModal.sections.editorFont')"
              :description="$t('settingsModal.sections.editorFontDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.editorFontSize')"
                :description="$t('settingsModal.editorFontSizeDesc')"
              >
                <SqSelect
                  v-model="settingsStore.editorFontSize"
                  :options="fontSizeOptions"
                  option-label="label"
                  option-value="value"
                  class="w-full max-w-[110px]"
                />
              </SettingsRow>

              <SettingsRow
                :label="$t('settingsModal.editorFontFamily')"
                :description="$t('settingsModal.editorFontFamilyDesc')"
              >
                <div class="w-full max-w-[360px]">
                  <FontFamilyPicker
                    v-model="settingsStore.editorFontFamily"
                    :options="EDITOR_FONT_FAMILY_OPTIONS"
                    :default-value="DEFAULT_EDITOR_FONT_FAMILY"
                    select-class="w-full !text-xs !bg-dark-900 !border-dark-700 font-mono"
                    preview
                  />
                </div>
              </SettingsRow>
            </SettingsSection>

            <SettingsSection
              :title="$t('settingsModal.sections.editorBehavior')"
              :description="$t('settingsModal.sections.editorBehaviorDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.editorWordWrap')"
                :description="$t('settingsModal.editorWordWrapDesc')"
              >
                <SqSelect
                  v-model="settingsStore.editorWordWrap"
                  :options="[
                    { label: '開啟 (On)', value: 'on' },
                    { label: '關閉 (Off)', value: 'off' },
                  ]"
                  option-label="label"
                  option-value="value"
                  class="w-full max-w-[130px]"
                />
              </SettingsRow>

              <SettingsRow
                :label="$t('settingsModal.editorTabSize')"
                :description="$t('settingsModal.editorTabSizeDesc')"
              >
                <SqSelect
                  v-model="settingsStore.editorTabSize"
                  :options="[
                    { label: '2 空格', value: 2 },
                    { label: '4 空格', value: 4 },
                  ]"
                  option-label="label"
                  option-value="value"
                  class="w-full max-w-[110px]"
                />
              </SettingsRow>

              <SettingsRow
                :label="$t('settingsModal.editorAutoCompletion')"
                :description="$t('settingsModal.editorAutoCompletionDesc')"
              >
                <SqSwitch v-model="settingsStore.editorAutoCompletion" />
              </SettingsRow>

              <SettingsRow
                :label="$t('settingsModal.editorKeywordsBold')"
                :description="$t('settingsModal.editorKeywordsBoldDesc')"
              >
                <SqSwitch v-model="settingsStore.editorKeywordsBold" />
              </SettingsRow>
            </SettingsSection>

            <SettingsSection
              :title="$t('settingsModal.sections.editorHighlight')"
              :description="$t('settingsModal.sections.editorHighlightDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.editorHighlightColor')"
                :description="$t('settingsModal.editorHighlightColorDesc')"
              >
                <div class="flex items-center space-x-2">
                  <input
                    type="color"
                    v-model="settingsStore.editorHighlightColor"
                    class="w-7 h-7 rounded border border-dark-700 bg-dark-900 cursor-pointer p-0.5"
                    title="選擇高亮顏色"
                  />
                  <SqInput
                    v-model="settingsStore.editorHighlightColor"
                    class="w-20 font-mono"
                    placeholder="#feffe0"
                  />
                </div>
              </SettingsRow>

              <EditorBackgroundSettings />
            </SettingsSection>
          </div>

          <!-- Tab 2: Query & Results Settings -->
          <div v-else-if="activeTab === 'results'" class="space-y-7">
            <SettingsSection
              :title="$t('settingsModal.sections.resultGrid')"
              :description="$t('settingsModal.sections.resultGridDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.gridFont')"
                :description="$t('settingsModal.gridFontDesc')"
              >
                <div class="w-full max-w-[320px]">
                  <FontFamilyPicker
                    v-model="settingsStore.gridFontFamily"
                    :options="GRID_FONT_FAMILY_OPTIONS"
                    :default-value="DEFAULT_GRID_FONT_FAMILY"
                    select-class="w-full !text-xs !bg-dark-900 !border-dark-700 font-mono"
                    preview
                  />
                </div>
              </SettingsRow>

              <SettingsRow
                :label="$t('settingsModal.gridFontSize')"
                :description="$t('settingsModal.gridFontSizeDesc')"
              >
                <SqSelect
                  v-model="settingsStore.gridFontSize"
                  :options="gridFontSizeOptions"
                  option-label="label"
                  option-value="value"
                  class="w-full max-w-[140px]"
                />
              </SettingsRow>
            </SettingsSection>

            <SettingsSection
              :title="$t('settingsModal.sections.resultTabs')"
              :description="$t('settingsModal.sections.resultTabsDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.maxResultTabs')"
                :description="$t('settingsModal.maxResultTabsDesc')"
              >
                <SqSelect
                  v-model="settingsStore.maxResultTabs"
                  :options="[
                    { label: '5 組', value: 5 },
                    { label: '10 組 (預設)', value: 10 },
                    { label: '15 組', value: 15 },
                    { label: '20 組', value: 20 },
                    { label: '30 組', value: 30 },
                    { label: '50 組', value: 50 },
                  ]"
                  option-label="label"
                  option-value="value"
                  class="w-full max-w-[140px]"
                />
              </SettingsRow>

              <SettingsRow
                :label="$t('settingsModal.accumulateResultTabs')"
                :description="$t('settingsModal.accumulateResultTabsDesc')"
              >
                <SqSwitch v-model="settingsStore.accumulateResultTabs" />
              </SettingsRow>
            </SettingsSection>

            <SettingsSection
              :title="$t('settingsModal.sections.dataRows')"
              :description="$t('settingsModal.sections.dataRowsDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.defaultMaxRows')"
                :description="$t('settingsModal.defaultMaxRowsDesc')"
              >
                <SqSelect
                  v-model="settingsStore.defaultMaxRows"
                  :options="[
                    { label: '1,000 筆', value: 1000 },
                    { label: '5,000 筆', value: 5000 },
                    { label: '10,000 筆 (預設)', value: 10000 },
                    { label: '50,000 筆', value: 50000 },
                    { label: '無限制 (No Limit)', value: null },
                  ]"
                  option-label="label"
                  option-value="value"
                  class="w-full max-w-[170px]"
                />
              </SettingsRow>
            </SettingsSection>

            <SettingsSection
              :title="$t('settingsModal.sections.dataChanges')"
              :description="$t('settingsModal.sections.dataChangesDesc')"
            >
              <SettingsRow
                :label="$t('settingsModal.showCommitSqlPreview')"
                :description="$t('settingsModal.showCommitSqlPreviewDesc')"
              >
                <SqSwitch v-model="settingsStore.showCommitSqlPreview" />
              </SettingsRow>
            </SettingsSection>
          </div>

      <!-- Tab 3: Table Filter -->
      <TableFilterTab v-else-if="activeTab === 'table_filter'" />

      <!-- Tab 4: AI Settings -->
      <AiSettingsTab v-else-if="activeTab === 'ai'" />

      <!-- Tab 5: About & Shortcuts -->
      <div v-else-if="activeTab === 'about'" class="space-y-4">
        <div class="bg-dark-900 border border-dark-750 p-4 rounded-md space-y-3">
          <div class="flex items-center space-x-3">
            <img
              :src="appIcon"
              alt="PuffSQL"
              class="w-10 h-10 rounded-md select-none flex-shrink-0"
              draggable="false"
            />
            <div>
              <div class="flex items-center space-x-2">
                <span class="font-bold text-dark-100 text-base">PuffSQL</span>
                <SqChip tone="neutral" :label="`v${appVersion}`" class="font-mono" />
              </div>
              <span class="text-xxs text-dark-400 font-mono">Lightweight & Fast SQL Client</span>
            </div>
          </div>
          <p class="text-xxs text-dark-400 leading-relaxed border-t border-dark-800 pt-2.5">
            {{ $t('settingsModal.aboutTitle') }}<br />
            {{ $t('settingsModal.aboutSubtitle') }}
          </p>
        </div>

        <!-- Keyboard Shortcuts -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-semibold text-dark-100 text-xs">{{ $t('settingsModal.shortcutsTitle') }}</span>
            <span class="text-xxs text-dark-500 font-mono">{{ $t('settingsModal.shortcutsSubtitle') }}</span>
          </div>

          <div class="grid grid-cols-2 gap-2.5 text-xxs font-mono">
            <!-- Group 1: 查詢執行與中斷 -->
            <div class="bg-dark-900 p-2.5 rounded border border-dark-800 space-y-2">
              <div class="text-xxs font-semibold text-ok flex items-center space-x-1.5 pb-1 border-b border-dark-800">
                <Play class="w-3 h-3"></Play>
                <span>{{ $t('settingsModal.shortcutGroupRun') }}</span>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutRunSelected') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-ok border border-dark-700">Ctrl + Enter</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutRunAll') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-ok border border-dark-700">Ctrl+Shift+Enter</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutCancel') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-danger border border-dark-700">Esc / Alt+Pause</kbd>
              </div>
            </div>

            <!-- Group 2: 分頁與檔案操作 -->
            <div class="bg-dark-900 p-2.5 rounded border border-dark-800 space-y-2">
              <div class="text-xxs font-semibold text-info flex items-center space-x-1.5 pb-1 border-b border-dark-800">
                <Folder class="w-3 h-3"></Folder>
                <span>{{ $t('settingsModal.shortcutGroupTabs') }}</span>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutNewTab') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-info border border-dark-700">Ctrl + N</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutSaveFile') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-info border border-dark-700">Ctrl + S</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutOpenFile') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-info border border-dark-700">Ctrl + O</kbd>
              </div>
            </div>

            <!-- Group 3: 編輯器輔助與格式化 -->
            <div class="bg-dark-900 p-2.5 rounded border border-dark-800 space-y-2">
              <div class="text-xxs font-semibold text-warn flex items-center space-x-1.5 pb-1 border-b border-dark-800">
                <FileCode class="w-3 h-3"></FileCode>
                <span>{{ $t('settingsModal.shortcutGroupEditor') }}</span>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutCompletion') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-warn border border-dark-700">Ctrl+Shift+A</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutFormat') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-warn border border-dark-700">Shift+Alt+F</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutAi') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-warn border border-dark-700">Ctrl + I</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutDuplicateLine') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-dark-200 border border-dark-700">Ctrl + D</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutFindReplace') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-dark-200 border border-dark-700">Ctrl+F / Ctrl+H</kbd>
              </div>
            </div>

            <!-- Group 4: 檢索、結果與圖表 -->
            <div class="bg-dark-900 p-2.5 rounded border border-dark-800 space-y-2">
              <div class="text-xxs font-semibold text-er flex items-center space-x-1.5 pb-1 border-b border-dark-800">
                <Search class="w-3 h-3"></Search>
                <span>{{ $t('settingsModal.shortcutGroupExplore') }}</span>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutQuickFinder') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-er border border-dark-700">Ctrl + P</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutCopyCell') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-structure border border-dark-700">Ctrl + C</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutSelectAllRows') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-structure border border-dark-700">Ctrl + A</kbd>
              </div>
              <div class="flex justify-between items-center py-0.5">
                <span class="text-dark-300">{{ $t('settingsModal.shortcutErDelete') }}</span>
                <kbd class="bg-dark-800 px-1.5 py-0.5 rounded text-dark-200 border border-dark-700">Del / Backspace</kbd>
              </div>
            </div>
          </div>
        </div>
      </div>
        </div>
      </div>
    </div>

    <!-- Modal Footer -->
    <template #footer>
      <div class="px-5 py-2.5 border-t border-dark-750 bg-dark-800 flex items-center justify-between w-full">
        <SqButton
          type="button"
          :label="$t('settingsModal.resetDefaults')"
          size="sm"
          tone="danger"
          @click="handleReset"
        />
        <SqButton
          type="button"
          :label="$t('settingsModal.done')"
          tone="accent"
          variant="solid"
          @click="$emit('close')"
        />
      </div>
    </template>
  </SqDialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import appIcon from '../../assets/puffsql_logo.png';
import SqDialog from '@/components/ui/SqDialog.vue';
import SqButton from '@/components/ui/SqButton.vue';
import {
  Check,
  FileCode,
  Filter,
  Folder,
  Info,
  Palette,
  Play,
  Search,
  Settings,
  Sparkles,
  Table2,
} from '@/components/icons';
import SqSelect from '@/components/ui/SqSelect.vue';
import SqSegmented from '@/components/ui/SqSegmented.vue';
import SqSwitch from '@/components/ui/SqSwitch.vue';
import SqCheckbox from '@/components/ui/SqCheckbox.vue';
import SqInput from '@/components/ui/SqInput.vue';
import SqChip from '@/components/ui/SqChip.vue';
import { useSettingsStore } from '@/stores/settingsStore';
import { useDialogWindow } from '@/composables/useDialogWindow';
import { SUPPORTED_LOCALES, type SupportedLocale } from '@/i18n';
import {
  GLOBAL_FONT_OPTIONS,
  DEFAULT_GLOBAL_FONT_FAMILY,
} from '@/services/themeManager';
import {
  DEFAULT_EDITOR_FONT_FAMILY,
  DEFAULT_GRID_FONT_FAMILY,
  EDITOR_FONT_FAMILY_OPTIONS,
  GRID_FONT_FAMILY_OPTIONS,
} from '@/data/fontOptions';
import FontFamilyPicker from '@/components/common/FontFamilyPicker.vue';
import TableFilterTab from './TableFilterTab.vue';
import AiSettingsTab from './settings/AiSettingsTab.vue';
import EditorBackgroundSettings from './settings/EditorBackgroundSettings.vue';
import SettingsRow from './settings/SettingsRow.vue';
import SettingsSection from './settings/SettingsSection.vue';

defineProps<{
  isOpen: boolean;
}>();

defineEmits<{
  (e: 'close'): void;
}>();

const { t } = useI18n();
/** Injected from package.json at build time, so the version is declared in exactly one place. */
const appVersion = __APP_VERSION__;
const settingsStore = useSettingsStore();
const dialogWindow = useDialogWindow();
const activeTab = ref<'theme' | 'editor' | 'results' | 'table_filter' | 'ai' | 'about'>('theme');

const tabs = computed(() => [
  {
    id: 'theme' as const,
    label: t('settingsModal.tabs.appearance'),
    desc: t('settingsModal.pageDesc.appearance'),
    icon: Palette,
  },
  {
    id: 'editor' as const,
    label: t('settingsModal.tabs.editor'),
    desc: t('settingsModal.pageDesc.editor'),
    icon: FileCode,
  },
  {
    id: 'results' as const,
    label: t('settingsModal.tabs.results'),
    desc: t('settingsModal.pageDesc.results'),
    icon: Table2,
  },
  {
    id: 'table_filter' as const,
    label: t('settingsModal.tabs.filter'),
    desc: t('settingsModal.pageDesc.table_filter'),
    icon: Filter,
  },
  {
    id: 'ai' as const,
    label: t('settingsModal.tabs.ai'),
    desc: t('settingsModal.pageDesc.ai'),
    icon: Sparkles,
  },
  {
    id: 'about' as const,
    label: t('settingsModal.tabs.about'),
    desc: t('settingsModal.pageDesc.about'),
    icon: Info,
  },
]);

const activeTabMeta = computed(
  () => tabs.value.find((tab) => tab.id === activeTab.value) ?? tabs.value[0]!
);

function handleLocaleChange(val: unknown) {
  if (val && typeof val === 'string') {
    settingsStore.setLocale(val as SupportedLocale);
  }
}


const globalFontSelectOptions = GLOBAL_FONT_OPTIONS.map(({ label, value }) => ({
  label,
  value,
}));

const selectedGlobalFont = computed(() =>
  GLOBAL_FONT_OPTIONS.find((option) => option.value === settingsStore.globalFontFamily)
);

function handleGlobalFontChange(fontFamily: unknown) {
  if (typeof fontFamily === 'string' && fontFamily.trim()) {
    settingsStore.setGlobalFontFamily(fontFamily);
  }
}

const fontSizeOptions = [
  { label: '12 px', value: 12 },
  { label: '13 px', value: 13 },
  { label: '14 px', value: 14 },
  { label: '15 px', value: 15 },
  { label: '16 px', value: 16 },
  { label: '18 px', value: 18 },
  { label: '20 px', value: 20 },
];

const uiFontSizeOptions = [
  { label: '13 px', value: 13 },
  { label: '14 px (預設)', value: 14 },
  { label: '15 px', value: 15 },
  { label: '16 px', value: 16 },
];

const gridFontSizeOptions = [
  { label: '11 px', value: 11 },
  { label: '12 px', value: 12 },
  { label: '13 px (預設)', value: 13 },
  { label: '14 px', value: 14 },
  { label: '15 px', value: 15 },
  { label: '16 px', value: 16 },
];

function handleUiFontSizeChange(val: unknown) {
  const size = typeof val === 'number' ? val : Number(val);
  if (Number.isFinite(size)) {
    settingsStore.setUiFontSize(size);
  }
}

function handleReset() {
  if (confirm(t('settingsModal.resetConfirm'))) {
    settingsStore.resetToDefaults();
  }
}
</script>
