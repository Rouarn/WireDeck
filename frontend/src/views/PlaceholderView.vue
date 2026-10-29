<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <section class="placeholder">
    <div class="placeholder-mark">W</div>
    <h2>{{ route.meta.title }}</h2>
    <p>{{ route.meta.description }}</p>
  </section>
</template>

<style scoped>
.placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 420px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius);
  background: var(--surface);
  color: var(--text-secondary);
}

.placeholder-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin-bottom: 4px;
  border-radius: 11px;
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 18px;
  font-weight: 700;
}

.placeholder h2 {
  margin: 0;
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
}

.placeholder p {
  margin: 0;
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>
