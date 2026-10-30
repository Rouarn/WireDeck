<script setup lang="ts" generic="T extends string">
defineProps<{
  options: { label: string; value: T }[]
  value: T
}>()

const emit = defineEmits<{
  (e: 'update:value', value: T): void
}>()
</script>

<template>
  <div class="segmented">
    <button
      v-for="item in options"
      :key="item.value"
      class="seg"
      :class="{ active: value === item.value }"
      type="button"
      @click="emit('update:value', item.value)"
    >
      {{ item.label }}
    </button>
  </div>
</template>

<style scoped>
.segmented {
  display: inline-flex;
  overflow: hidden;
  border: 1px solid var(--border-strong);
  border-radius: 7px;
  background: var(--surface);
}

.seg {
  height: 32px;
  padding: 0 16px;
  border: none;
  border-right: 1px solid var(--border);
  background: transparent;
  color: var(--text-secondary);
  font-family: inherit;
  font-size: 12px;
  line-height: 32px;
  cursor: pointer;
}

.seg:last-child {
  border-right: none;
}

.seg:hover {
  color: var(--text);
}

.seg.active {
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-weight: 600;
}
</style>
