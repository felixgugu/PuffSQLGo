<template>
  <SqDialog
    :visible="open"
    :header="title"
    :class="dialogWindow.marker"
    class="w-[760px] max-w-[94vw]"
    content-class="!p-0"
    @show="dialogWindow.onShow"
    @hide="dialogWindow.onHide"
    @update:visible="value => !value && close()"
  >
    <div class="p-4 space-y-4 text-xs">
      <div class="rounded border border-dark-700 bg-dark-850 px-3 py-2 font-mono text-dark-300">
        {{ schema }}.{{ table }}<span v-if="column?.name" class="text-dark-500"> / {{ column.name }}</span>
      </div>

      <div v-if="operation !== 'drop'" class="grid grid-cols-2 gap-3">
        <label class="space-y-1">
          <span class="text-dark-300">{{ $t('columnEditor.columnName') }} *</span>
          <SqInput v-model="form.columnName" :disabled="busy" @update:model-value="resetPreview" />
        </label>
        <label class="space-y-1">
          <span class="text-dark-300">{{ $t('columnEditor.dataType') }} *</span>
          <SqSelect
            v-model="form.dataType"
            :options="columnTypes"
            size="md"
            :aria-label="$t('columnEditor.dataType')"
            @update:model-value="onTypeChange"
          />
        </label>
        <label v-if="needsLength" class="space-y-1">
          <span class="text-dark-300">{{ $t('columnEditor.length') }} *</span>
          <SqInput
            :model-value="form.length"
            type="number"
            :placeholder="$t('columnEditor.lengthHint')"
            @update:model-value="value => { form.length = Number(value); resetPreview(); }"
          />
        </label>
        <label v-if="needsPrecision" class="space-y-1">
          <span class="text-dark-300">{{ $t('columnEditor.precision') }} *</span>
          <SqInput
            :model-value="form.precision"
            type="number"
            @update:model-value="value => { form.precision = Number(value); resetPreview(); }"
          />
        </label>
        <label v-if="needsPrecision" class="space-y-1">
          <span class="text-dark-300">{{ $t('columnEditor.scale') }} *</span>
          <SqInput
            :model-value="form.scale"
            type="number"
            @update:model-value="value => { form.scale = Number(value); resetPreview(); }"
          />
        </label>
        <label v-if="needsScale" class="space-y-1">
          <span class="text-dark-300">{{ $t('columnEditor.timeScale') }} *</span>
          <SqInput
            :model-value="form.scale"
            type="number"
            @update:model-value="value => { form.scale = Number(value); resetPreview(); }"
          />
        </label>
        <label class="space-y-1 col-span-2">
          <span class="text-dark-300">{{ $t('columnEditor.defaultValue') }}</span>
          <SqInput
            v-model="form.defaultExpression"
            :placeholder="$t('columnEditor.defaultHint')"
            @update:model-value="resetPreview"
          />
        </label>
        <label class="space-y-1 col-span-2">
          <span class="text-dark-300">{{ $t('columnEditor.description') }}</span>
          <SqInput
            v-model="form.description"
            :placeholder="$t('columnEditor.descriptionHint')"
            maxlength="4000"
            @update:model-value="resetPreview"
          />
        </label>
        <div class="col-span-2 flex items-center justify-between rounded border border-dark-700 px-3 py-2">
          <div>
            <div class="text-dark-200">{{ $t('columnEditor.allowNull') }}</div>
            <div class="text-xxs text-dark-500 mt-0.5">{{ $t('columnEditor.allowNullHint') }}</div>
          </div>
          <SqSwitch v-model="form.isNullable" :disabled="busy" @update:model-value="resetPreview" />
        </div>
      </div>

      <div v-if="operation === 'drop'" class="rounded border border-rose-300 dark:border-rose-900 bg-rose-50 dark:bg-rose-950/30 p-3 text-danger">
        {{ $t('columnEditor.dropWarning') }} <strong class="font-mono">{{ column?.name }}</strong>
      </div>

      <div v-if="error" class="rounded border border-rose-300 bg-rose-50 dark:bg-rose-950/30 p-3 text-danger">
        {{ error }}
      </div>

      <div v-if="preview" class="space-y-3">
        <section v-if="preview.blockingIssues.length" class="rounded border border-rose-300 dark:border-rose-900 p-3">
          <div class="font-semibold text-danger mb-1">{{ $t('columnEditor.blockingTitle') }}</div>
          <ul class="list-disc pl-4 space-y-1 text-danger">
            <li v-for="item in preview.blockingIssues" :key="item.code + item.message">{{ item.message }}</li>
          </ul>
        </section>
        <section v-if="preview.warnings.length" class="rounded border border-amber-300 dark:border-amber-900 p-3">
          <div class="font-semibold text-warn mb-1">{{ $t('columnEditor.warningTitle') }}</div>
          <ul class="list-disc pl-4 space-y-1 text-warn">
            <li v-for="item in preview.warnings" :key="item.code + item.message">{{ item.message }}</li>
          </ul>
        </section>
        <section v-if="preview.dependencies.length" class="rounded border border-dark-700 p-3">
          <div class="font-semibold text-dark-100 mb-1">{{ $t('columnEditor.dependencyTitle') }}</div>
          <div class="font-mono text-xxs text-dark-400 space-y-0.5 max-h-24 overflow-y-auto">
            <div v-for="item in preview.dependencies" :key="item.kind + item.name">
              {{ item.kind }} · {{ item.name }} · {{ item.detail }}
            </div>
          </div>
        </section>
        <section class="rounded border border-dark-700 overflow-hidden">
          <div class="h-7 px-3 flex items-center bg-dark-800 text-xxs font-semibold text-dark-300">
            {{ $t('columnEditor.sqlPreview') }}
          </div>
          <div class="h-52 bg-dark-900">
            <SqlCodeViewer :code="preview.sql" />
          </div>
        </section>
      </div>
    </div>

    <template #footer>
      <SqButton :label="$t('columnEditor.cancel')" :disabled="busy" @click="close" />
      <SqButton
        v-if="operation !== 'drop' && !preview"
        :label="$t('columnEditor.preview')"
        tone="accent"
        variant="solid"
        :loading="busy"
        @click="loadPreview"
      />
      <SqButton
        v-else
        :label="operation === 'drop' ? $t('columnEditor.drop') : $t('columnEditor.execute')"
        :tone="operation === 'drop' ? 'danger' : 'accent'"
        variant="solid"
        :loading="busy"
        :disabled="!!preview?.blockingIssues.length"
        @click="applyChange"
      />
    </template>
  </SqDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import SqButton from '@/components/ui/SqButton.vue';
import SqDialog from '@/components/ui/SqDialog.vue';
import SqInput from '@/components/ui/SqInput.vue';
import SqSelect from '@/components/ui/SqSelect.vue';
import SqSwitch from '@/components/ui/SqSwitch.vue';
import SqlCodeViewer from '@/components/common/SqlCodeViewer.vue';
import { useDialogWindow } from '@/composables/useDialogWindow';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { columnChangeService } from '@/services/columnChangeService';
import type { ColumnItem } from '@/types/schema';
import type { ColumnChangeOperation, ColumnChangePreview, ColumnChangeRequest } from '@/types/columnChange';

const props = defineProps<{
  open: boolean;
  connectionId: string;
  database: string;
  schema: string;
  table: string;
  operation: ColumnChangeOperation;
  column?: ColumnItem | null;
}>();
const { t } = useI18n();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'applied'): void;
}>();

const dialogWindow = useDialogWindow();
const workspaceStore = useWorkspaceStore();
const busy = ref(false);
const error = ref('');
const preview = ref<ColumnChangePreview | null>(null);

const columnTypes = [
  'bit', 'tinyint', 'smallint', 'int', 'bigint', 'decimal', 'numeric', 'money',
  'smallmoney', 'float', 'real', 'date', 'time', 'datetime', 'datetime2',
  'smalldatetime', 'datetimeoffset', 'char', 'varchar', 'nchar', 'nvarchar',
  'binary', 'varbinary', 'uniqueidentifier', 'xml',
];

const form = reactive({
  columnName: '',
  dataType: 'nvarchar',
  length: 100,
  precision: 18,
  scale: 0,
  isNullable: true,
  defaultExpression: '',
  description: '',
});

const title = computed(() => {
  if (props.operation === 'add') return t('columnEditor.addTitle');
  if (props.operation === 'alter') return t('columnEditor.alterTitle');
  return t('columnEditor.dropTitle');
});
const needsLength = computed(() => ['char', 'varchar', 'nchar', 'nvarchar', 'binary', 'varbinary'].includes(form.dataType));
const needsPrecision = computed(() => ['decimal', 'numeric'].includes(form.dataType));
const needsScale = computed(() => ['time', 'datetime2', 'datetimeoffset'].includes(form.dataType));

function initialize() {
  const column = props.column;
  form.columnName = column?.name ?? '';
  form.dataType = column?.dataType ?? 'nvarchar';
  form.length = column?.maxLength ?? (needsLength.value ? 100 : 0);
  form.precision = column?.precision ?? 18;
  form.scale = column?.scale ?? 0;
  form.isNullable = column?.isNullable ?? true;
  form.defaultExpression = column?.defaultDefinition ?? '';
  form.description = column?.description ?? '';
  preview.value = null;
  error.value = '';
  if (props.open && props.operation === 'drop') void loadPreview();
}

watch(() => [props.open, props.operation, props.column] as const, initialize, { immediate: true });

function onTypeChange() {
  if (needsLength.value && (!form.length || form.length === 0)) form.length = 100;
  if (needsPrecision.value && (!form.precision || form.precision === 0)) form.precision = 18;
  resetPreview();
}

function resetPreview() {
  preview.value = null;
  error.value = '';
}

function buildRequest(): ColumnChangeRequest {
  const originalName = props.operation === 'add' ? undefined : props.column?.name;
  return {
    connectionId: props.connectionId,
    database: props.database,
    schema: props.schema,
    table: props.table,
    operation: props.operation,
    originalName,
    columnName: props.operation === 'drop' ? (props.column?.name ?? '') : form.columnName.trim(),
    dataType: form.dataType,
    length: needsLength.value ? form.length : null,
    precision: needsPrecision.value ? form.precision : null,
    scale: needsPrecision.value || needsScale.value ? form.scale : null,
    isNullable: form.isNullable,
    defaultExpression: form.defaultExpression.trim() || null,
    description: form.description.trim() || null,
  };
}

async function loadPreview() {
  if (props.operation !== 'drop' && !form.columnName.trim()) {
    error.value = t('columnEditor.nameRequired');
    return;
  }
  busy.value = true;
  error.value = '';
  try {
    preview.value = await columnChangeService.preview(buildRequest());
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

async function applyChange() {
  if (!preview.value || preview.value.blockingIssues.length) return;
  busy.value = true;
  error.value = '';
  try {
    await columnChangeService.apply(buildRequest(), preview.value.fingerprint);
    workspaceStore.showToast(t('columnEditor.success'), 'success', 2000);
    emit('applied');
    emit('close');
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

function close() {
  if (!busy.value) emit('close');
}
</script>
