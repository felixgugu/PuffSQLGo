<template>
  <div class="settings-row">
    <label class="settings-row__label font-medium text-dark-100">{{ label }}</label>
    <div class="settings-row__control">
      <slot />
    </div>
    <p v-if="description || $slots.description" class="settings-row__desc text-xxs text-dark-500">
      <slot name="description">{{ description }}</slot>
    </p>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string;
    description?: string;
  }>(),
  { description: '' }
);
</script>

<style scoped>
/*
 * A narrow, fixed label column keeps the form compact: the control column starts at the same
 * x-position on every row, instead of being pushed to the far right of a wide dialog. The
 * description is its own row spanning both columns, so it can never widen the label column.
 */
.settings-row {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr);
  column-gap: 24px;
  row-gap: 2px;
  align-items: start;
  padding: 12px 0;
}

.settings-row__label {
  min-width: 0;
  line-height: 1.45;
}

.settings-row__control {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  min-width: 0;
}

.settings-row__desc {
  grid-column: 1 / -1;
  max-width: 560px;
  margin: 0;
  line-height: 1.5;
}
</style>
