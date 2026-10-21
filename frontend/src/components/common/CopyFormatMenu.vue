<template>
  <div ref="anchorRef" class="relative">
    <SqButton
      type="button"
      :label="copied ? $t('common.copied') : $t('copyFormats.copy')"
      size="sm"
      variant="outline"
      :aria-expanded="open"
      :aria-label="$t('copyFormats.ariaLabel')"
      @click="open = !open"
    >
      <template #icon>
        <Copy class="w-3.5 h-3.5" />
        <ChevronDown class="w-3 h-3" />
      </template>
    </SqButton>
    <SqPopover
      v-model="open"
      :anchor="anchorRef"
      align="end"
      min-width="12rem"
      role="menu"
      :aria-label="$t('copyFormats.ariaLabel')"
    >
      <button
        v-for="format in formats"
        :key="format.id"
        type="button"
        role="menuitem"
        class="w-full text-left px-3 py-2 text-xs text-dark-200 hover:bg-dark-750 hover:text-dark-100 flex items-center justify-between gap-4"
        @click="choose(format.id)"
      >
        <span>{{ format.label }}</span>
        <span class="text-xxs text-dark-500">{{ format.hint }}</span>
      </button>
    </SqPopover>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { ChevronDown, Copy } from '@/components/icons';
import SqButton from '@/components/ui/SqButton.vue';
import SqPopover from '@/components/ui/SqPopover.vue';
import type { CopyFormat } from '@/types/copyFormat';

withDefaults(defineProps<{ copied?: boolean }>(), { copied: false });
const emit = defineEmits<{ (e: 'copy', format: CopyFormat): void }>();
const { t } = useI18n();
const open = ref(false);
const anchorRef = ref<HTMLElement | null>(null);

const formats = computed(() => [
  { id: 'tsv' as const, label: t('copyFormats.tsv'), hint: 'Excel' },
  { id: 'csv' as const, label: t('copyFormats.csv'), hint: '.csv' },
  { id: 'json' as const, label: t('copyFormats.json'), hint: 'JSON' },
  { id: 'markdown' as const, label: t('copyFormats.markdown'), hint: 'MD' },
]);

function choose(format: CopyFormat) {
  open.value = false;
  emit('copy', format);
}
</script>
