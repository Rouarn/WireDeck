<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { NButton, NIcon, NInput, NSelect, useMessage } from 'naive-ui'
import { PlayOutline, StopOutline } from '@vicons/ionicons5'
import { CopyText, ListApps, StartLogcat, StopLogcat } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import VirtualList from '../components/VirtualList.vue'
import { useDevicesStore } from '../stores/devices'

interface LogcatBatchPayload {
  streamId: string
  lines: string[]
}

interface LogLine {
  level: string
  text: string
}

const store = useDevicesStore()
const message = useMessage()

const running = ref(false)
const streamId = ref('')
const lines = ref<LogLine[]>([])
const level = ref('WA')
const keyword = ref('')
const paused = ref(false)

const apps = ref<model.AppSummary[]>([])
const selectedPkg = ref('')
const crashActive = ref(false)
const crashBuffer = ref<string[]>([])

const listRef = ref<{ scrollToBottom: () => void } | null>(null)

const appOptions = computed(() => [
  { label: '全部应用', value: '' },
  ...apps.value.map((app) => ({
    label: app.label ? `${app.label} (${app.package})` : app.package,
    value: app.package,
  })),
])

const levelOptions = [
  { label: '警告与错误', value: 'WA' },
  { label: '全部级别', value: 'ALL' },
  { label: 'ERROR', value: 'E' },
  { label: 'WARN', value: 'W' },
  { label: 'INFO', value: 'I' },
  { label: 'DEBUG', value: 'D' },
  { label: 'VERBOSE', value: 'V' },
]

const visibleLines = computed(() => {
  let result = lines.value
  if (level.value === 'WA') {
    result = result.filter((line) => ['W', 'E', 'F'].includes(line.level))
  } else if (level.value !== 'ALL') {
    result = result.filter((line) => line.level === level.value)
  }
  if (keyword.value.trim()) {
    const key = keyword.value.trim()
    result = result.filter((line) => line.text.includes(key))
  }
  return result
})

function parseLine(raw: string): LogLine {
  // 06-30 12:00:00.000  1234  5678 I Tag: message
  const match = raw.match(/^\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\.\d+\s+\d+\s+\d+\s+([VDIWEF])\s+/)
  return { level: match ? match[1] : '?', text: raw }
}

function trackCrash(raw: string) {
  if (raw.includes('FATAL EXCEPTION') || raw.includes('ANR in')) {
    crashActive.value = true
    crashBuffer.value = [raw]
    return
  }
  if (crashActive.value && crashBuffer.value.length < 60) {
    crashBuffer.value.push(raw)
  }
}

function onBatch(p: LogcatBatchPayload) {
  if (!running.value || p.streamId !== streamId.value || paused.value) {
    return
  }

  for (const raw of p.lines) {
    lines.value.push(parseLine(raw))
    trackCrash(raw)
  }
  if (lines.value.length > 2000) {
    lines.value.splice(0, lines.value.length - 2000)
  }
}

// 新日志到达后贴到底部
watch(
  () => visibleLines.value.length,
  () => {
    if (!paused.value) {
      listRef.value?.scrollToBottom()
    }
  },
)

async function copyCrash() {
  try {
    await CopyText(crashBuffer.value.join('\n'))
    message.success('崩溃日志已复制')
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

function dismissCrash() {
  crashActive.value = false
  crashBuffer.value = []
}

async function loadApps() {
  if (!store.current) {
    apps.value = []
    return
  }
  try {
    apps.value = await ListApps(store.current.serial, 'user')
  } catch {
    apps.value = []
  }
}

async function startLog() {
  if (!store.current) {
    message.warning('请先在「设备」页选择设备')
    return
  }
  if (running.value) {
    return
  }

  lines.value = []
  crashActive.value = false
  crashBuffer.value = []
  try {
    streamId.value = await StartLogcat(store.current.serial, selectedPkg.value || '')
    running.value = true
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function stopLog() {
  if (streamId.value) {
    await StopLogcat(streamId.value)
  }
  running.value = false
  streamId.value = ''
}

function clearLog() {
  lines.value = []
  crashActive.value = false
  crashBuffer.value = []
}

let unlisten: (() => void) | null = null

onMounted(() => {
  unlisten = EventsOn('logcat:batch', (p: LogcatBatchPayload) => onBatch(p))
  void loadApps()
})

onBeforeUnmount(() => {
  if (streamId.value) {
    void StopLogcat(streamId.value)
  }
  if (unlisten) {
    unlisten()
    unlisten = null
  }
})
</script>

<template>
  <div class="logs">
    <div class="toolbar">
      <span class="count">{{ visibleLines.length }} / {{ lines.length }} 行</span>
      <div class="spacer"></div>
      <NButton size="small" type="primary" :disabled="running || !store.current" @click="startLog">
        <template #icon><NIcon><PlayOutline /></NIcon></template>
        开始
      </NButton>
      <NButton size="small" :disabled="!running" @click="stopLog">
        <template #icon><NIcon><StopOutline /></NIcon></template>
        停止
      </NButton>
      <NButton size="small" quaternary @click="paused = !paused">{{ paused ? '继续' : '暂停' }}</NButton>
      <NButton size="small" quaternary @click="clearLog">清空</NButton>
    </div>

    <div class="filters">
      <NSelect
        v-model:value="selectedPkg"
        size="small"
        filterable
        clearable
        placeholder="全部应用"
        :options="appOptions"
        style="width: 240px"
      />
      <NSelect v-model:value="level" size="small" :options="levelOptions" style="width: 140px" />
      <NInput v-model:value="keyword" size="small" clearable placeholder="关键字过滤" style="width: 200px" />
      <span class="hint">最多保留最近 2000 行 · 虚拟滚动</span>
    </div>

    <div v-if="crashActive" class="crash-bar">
      <span>检测到崩溃 / ANR，已收集 {{ crashBuffer.length }} 行日志</span>
      <div class="spacer"></div>
      <NButton size="tiny" type="primary" @click="copyCrash">复制日志</NButton>
      <NButton size="tiny" quaternary @click="dismissCrash">忽略</NButton>
    </div>

    <div class="log-box">
      <VirtualList ref="listRef" :items="visibleLines" :item-height="19">
        <template #default="{ item }">
          <div class="log-line" :class="`level-${item.level}`" :title="item.text">{{ item.text }}</div>
        </template>
      </VirtualList>
      <div v-if="!visibleLines.length" class="log-empty">
        {{ running ? '等待日志…' : '点击「开始」读取实时日志' }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.logs {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 96px);
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.toolbar .count {
  color: var(--text-tertiary);
  font-size: 11px;
}

.spacer {
  flex: 1;
}

.filters {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.filters .hint {
  color: var(--text-tertiary);
  font-size: 11px;
}

.crash-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  padding: 8px 12px;
  border: 1px solid var(--danger-soft);
  border-radius: var(--radius-sm);
  background: var(--danger-soft);
  color: var(--danger);
  font-size: 12px;
}

.log-box {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface-inset);
}

.log-line {
  height: 19px;
  padding: 0 12px;
  overflow: hidden;
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 19px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.level-V {
  color: var(--text-tertiary);
}

.level-D {
  color: var(--text-secondary);
}

.level-I {
  color: var(--primary-ink);
}

.level-W {
  color: var(--warning);
}

.level-E,
.level-F {
  color: var(--danger);
}

.log-empty {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: var(--text-tertiary);
  font-size: 12px;
}
</style>
