<template>
  <SqDialog
    :visible="open"
    :header="$t('excelTableExport.title')"
    :class="dialogWindow.marker"
    class="w-[620px] max-w-[92vw]"
    content-class="!p-0"
    @show="dialogWindow.onShow"
    @hide="dialogWindow.onHide"
    @update:visible="value => !value && emit('close')"
  >
    <div class="p-4 space-y-3 text-xs">
      <div class="flex items-center gap-2">
        <SqInput v-model="search" :placeholder="$t('excelTableExport.search')" size="sm" class="flex-1" />
        <SqButton size="sm" :label="$t('excelTableExport.selectAll')" @click="selectAll" />
        <SqButton size="sm" :label="$t('excelTableExport.selectNone')" @click="selectedKeys = []" />
      </div>
      <div class="text-xxs text-dark-500">
        {{ $t('excelTableExport.hint', { count: filteredTables.length }) }}
      </div>
      <div v-if="loading" class="py-10 flex items-center justify-center gap-2 text-dark-400">
        <RotateCw class="w-4 h-4 animate-spin text-accent" />
        {{ $t('excelTableExport.loading') }}
      </div>
      <div v-else-if="error" class="rounded border border-rose-300 dark:border-rose-900 p-3 text-danger">
        {{ error }}
      </div>
      <div v-else class="max-h-[52vh] overflow-y-auto rounded border border-dark-700 divide-y divide-dark-750">
        <label
          v-for="table in filteredTables"
          :key="tableKey(table)"
          class="flex items-center gap-2 px-3 py-2 hover:bg-dark-750/60 cursor-pointer"
        >
          <input
            type="checkbox"
            :checked="selectedKeys.includes(tableKey(table))"
            @change="toggle(table)"
          />
          <Table2 class="w-3.5 h-3.5 text-accent flex-shrink-0" />
          <span class="text-dark-400 font-mono">{{ table.schema }}.</span>
          <span class="text-dark-100 truncate">{{ table.name }}</span>
        </label>
        <div v-if="filteredTables.length === 0" class="py-8 text-center text-dark-500">
          {{ $t('excelTableExport.empty') }}
        </div>
      </div>
    </div>
    <template #footer>
      <span class="mr-auto text-xxs text-dark-400">
        {{ $t('excelTableExport.selected', { count: selectedKeys.length }) }}
      </span>
      <SqButton :label="$t('common.cancel')" @click="emit('close')" />
      <SqButton
        :label="$t('excelTableExport.export')"
        tone="accent"
        variant="solid"
        :disabled="selectedKeys.length === 0 || loading"
        @click="confirm"
      />
    </template>
  </SqDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { RotateCw, Table2 } from '@/components/icons';
import SqButton from '@/components/ui/SqButton.vue';
import SqDialog from '@/components/ui/SqDialog.vue';
import SqInput from '@/components/ui/SqInput.vue';
import { useDialogWindow } from '@/composables/useDialogWindow';
import { schemaService } from '@/services/schemaService';
import type { TableItem } from '@/types/schema';
import type { ExcelTableRef } from '@/types/excelExport';

const props = defineProps<{
  open: boolean;
  connectionId: string;
  database: string;
}>();
const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'export', tables: ExcelTableRef[]): void;
}>();

const dialogWindow = useDialogWindow();
const tables = ref<TableItem[]>([]);
const selectedKeys = ref<string[]>([]);
const search = ref('');
const loading = ref(false);
const error = ref('');

const filteredTables = computed(() => {
  const needle = search.value.trim().toLowerCase();
  return tables.value.filter((table) => {
    if (table.kind !== 'BASE TABLE') return false;
    return !needle || `${table.schema}.${table.name}`.toLowerCase().includes(needle);
  });
});

watch(
  () => [props.open, props.connectionId, props.database] as const,
  async ([open]) => {
    if (!open) return;
    search.value = '';
    selectedKeys.value = [];
    loading.value = true;
    error.value = '';
    try {
      tables.value = await schemaService.getTables(props.connectionId, props.database);
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  },
  { immediate: true }
);

function tableKey(table: TableItem) {
  return `${table.schema}\u0000${table.name}`;
}

function toggle(table: TableItem) {
  const key = tableKey(table);
  selectedKeys.value = selectedKeys.value.includes(key)
    ? selectedKeys.value.filter((item) => item !== key)
    : [...selectedKeys.value, key];
}

function selectAll() {
  selectedKeys.value = filteredTables.value.map(tableKey);
}

function confirm() {
  const selected = new Set(selectedKeys.value);
  emit(
    'export',
    tables.value
      .filter((table) => selected.has(tableKey(table)))
      .map((table) => ({ schema: table.schema, name: table.name }))
  );
}
</script>
