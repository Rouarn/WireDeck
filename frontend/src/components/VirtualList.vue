<script setup lang="ts" generic="T">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * 固定行高虚拟列表。
 * 容器高度由父级决定（height: 100%），仅渲染视口内 + overscan 条数据。
 */
const props = withDefaults(
  defineProps<{
    items: readonly T[]
    itemHeight: number
    overscan?: number
  }>(),
  { overscan: 6 },
)

const container = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportHeight = ref(0)

const totalHeight = computed(() => props.items.length * props.itemHeight)

const startIndex = computed(() =>
  Math.max(0, Math.floor(scrollTop.value / props.itemHeight) - props.overscan),
)

const endIndex = computed(() =>
  Math.min(
    props.items.length,
    Math.ceil((scrollTop.value + viewportHeight.value) / props.itemHeight) + props.overscan,
  ),
)

const offset = computed(() => startIndex.value * props.itemHeight)
const visible = computed(() => props.items.slice(startIndex.value, endIndex.value))

let frame = 0

function onScroll(event: Event) {
  const target = event.target as HTMLElement
  if (frame) {
    return
  }
  frame = requestAnimationFrame(() => {
    frame = 0
    scrollTop.value = target.scrollTop
  })
}

function scrollToBottom() {
  const el = container.value
  if (el) {
    el.scrollTop = el.scrollHeight
    scrollTop.value = el.scrollTop
  }
}

defineExpose({ scrollToBottom })

let observer: ResizeObserver | null = null

onMounted(() => {
  const el = container.value
  if (!el) {
    return
  }
  viewportHeight.value = el.clientHeight
  observer = new ResizeObserver(() => {
    viewportHeight.value = el.clientHeight
  })
  observer.observe(el)
})

onBeforeUnmount(() => {
  if (frame) {
    cancelAnimationFrame(frame)
  }
  observer?.disconnect()
})
</script>

<template>
  <div ref="container" class="relative h-full overflow-y-auto overflow-x-hidden" @scroll.passive="onScroll">
    <div class="relative w-full" :style="{ height: `${totalHeight}px` }">
      <div class="absolute inset-x-0 top-0" :style="{ transform: `translateY(${offset}px)` }">
        <div
          v-for="(item, index) in visible"
          :key="startIndex + index"
          class="overflow-hidden"
          :style="{ height: `${itemHeight}px` }"
        >
          <slot :item="item" :index="startIndex + index" />
        </div>
      </div>
    </div>
  </div>
</template>
