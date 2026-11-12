<template>
  <section class="settings-section">
    <header class="settings-section__header">
      <h3 class="settings-section__title">{{ title }}</h3>
      <p v-if="description" class="settings-section__desc">{{ description }}</p>
    </header>
    <div class="settings-section__body divide-y divide-dark-750/70">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    description?: string;
  }>(),
  { description: '' }
);
</script>

<style scoped>
.settings-section {
  min-width: 0;
}

.settings-section__header {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-bottom: 4px;
}

.settings-section__title {
  margin: 0;
  color: rgb(var(--color-dark-100));
  font-family: var(--app-font-sans);
  font-size: var(--app-font-size-secondary);
  font-weight: 600;
  line-height: 1.4;
}

.settings-section__desc {
  margin: 0;
  color: rgb(var(--color-dark-500));
  font-size: var(--app-font-size-caption);
  line-height: 1.5;
}

/* Rows own their vertical padding; the section only draws the hairlines between them. */
.settings-section__body {
  display: block;
}
</style>
