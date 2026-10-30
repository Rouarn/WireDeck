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
  <div class="inline-flex overflow-hidden rounded-[7px] border border-line-strong bg-surface">
    <button
      v-for="item in options"
      :key="item.value"
      type="button"
      class="h-8 cursor-pointer border-0 border-r border-line px-4 text-xs leading-8 last:border-r-0"
      :class="value === item.value ? 'bg-brand-soft font-600 text-brand-ink' : 'text-fg2 hover:text-fg'"
      @click="emit('update:value', item.value)"
    >
      {{ item.label }}
    </button>
  </div>
</template>
