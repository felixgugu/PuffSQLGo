<template>
  <div
    class="result-set-tabs"
    role="tablist"
    :aria-label="$t('results.resultSets')"
    @keydown="onKeydown"
  >
    <button
      v-for="(set, index) in sets"
      :key="`result-set-tab-${index}`"
      type="button"
      role="tab"
      :id="tabIdFor(index)"
      :aria-controls="panelId"
      :aria-selected="activeIndex === index"
      :tabindex="activeIndex === index ? 0 : -1"
      class="result-set-tab"
      :class="{ 'is-active': activeIndex === index }"
      @click="$emit('select', index)"
    >
      <Table2 class="w-3.5 h-3.5 flex-shrink-0" />
      <span class="truncate">
        {{
          $t('results.resultSetTabLabel', {
            index: index + 1,
            count: (set.rowCount ?? set.rows.length).toLocaleString(),
          })
        }}
      </span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { nextTick } from 'vue';
import { Table2 } from '@/components/icons';
import type { ResultSet } from '@/types/query';

const props = defineProps<{
  sets: ResultSet[];
  activeIndex: number;
  resultTabId?: string | null;
  panelId: string;
}>();

const emit = defineEmits<{
  (event: 'select', index: number): void;
}>();

function tabIdFor(index: number): string {
  return `result-set-tab-${props.resultTabId ?? 'orphan'}-${index}`;
}

function onKeydown(event: KeyboardEvent) {
  const keys = ['ArrowRight', 'ArrowLeft', 'Home', 'End'];
  if (!keys.includes(event.key) || props.sets.length === 0) return;

  let nextIndex = props.activeIndex;
  if (event.key === 'ArrowRight') nextIndex = (nextIndex + 1) % props.sets.length;
  if (event.key === 'ArrowLeft') {
    nextIndex = (nextIndex - 1 + props.sets.length) % props.sets.length;
  }
  if (event.key === 'Home') nextIndex = 0;
  if (event.key === 'End') nextIndex = props.sets.length - 1;

  event.preventDefault();
  if (nextIndex !== props.activeIndex) {
    // The emitted parent update is synchronous, so the next DOM tick can focus the new tab.
    emit('select', nextIndex);
  }
  nextTick(() => {
    document.getElementById(tabIdFor(nextIndex))?.focus();
  });
}
</script>

<style scoped>
.result-set-tabs {
  display: flex;
  align-items: flex-end;
  min-height: 31px;
  padding: 3px 6px 0;
  gap: 2px;
  overflow-x: auto;
  overflow-y: hidden;
  background: rgb(var(--color-dark-850));
  border-bottom: 1px solid rgb(var(--color-dark-700));
}

.result-set-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex: 0 0 auto;
  max-width: 260px;
  height: 27px;
  padding: 0 9px;
  border: 1px solid transparent;
  border-bottom: 0;
  border-radius: 4px 4px 0 0;
  background: transparent;
  color: rgb(var(--color-dark-400));
  font-family: var(--app-font-sans);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background-color 120ms ease, border-color 120ms ease, color 120ms ease;
}

.result-set-tab:hover {
  background: rgb(var(--color-dark-800));
  color: rgb(var(--color-dark-100));
}

.result-set-tab.is-active {
  background: rgb(var(--color-dark-900));
  border-color: rgb(var(--color-dark-700));
  color: rgb(var(--color-accent));
}

.result-set-tab:focus-visible {
  outline: 2px solid rgb(var(--color-accent));
  outline-offset: -2px;
}
</style>
