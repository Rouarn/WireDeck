<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { NButton, NIcon, NInput, NRadioButton, NRadioGroup, NSelect, NSpin, useMessage } from 'naive-ui'
import { PlayOutline, StopOutline, TerminalOutline } from '@vicons/ionicons5'
import { ExecShell, StartLogcat, StopLogcat } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
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

const mode = ref<'logcat' | 'shell'>('logcat')

const running = ref(false)
const streamId = ref('')
const lines = ref<LogLine[]>([])
const level = ref('ALL')
const keyword = ref('')
const paused = ref(false)
const logBox = ref<HTMLElement | null>(null)

const levelOptions = [
  { label: '全部级别', value: 'ALL' },
  { label: 'VERBOSE', value: 'V' },
  { label: 'DEBUG', value: 'D' },
  { label: 'INFO', value: 'I' },
  { label: 'WARN', value: 'W' },
  { label: 'ERROR', value: 'E' },
]

const visibleLines = computed(() => {
  let result = lines.value
  if (level.value !== 'ALL') {
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

function onBatch(p: LogcatBatchPayload) {
  if (!running.value || p.streamId !== streamId.value) {
    return
  }
  if (paused.value) {
    return
  }

  for (const raw of p.lines) {
    lines.value.push(parseLine(raw))
  }
  if (lines.value.length > 1000) {
    lines.value.splice(0, lines.value.length - 1000)
  }
  scheduleScroll()
}

let scrollQueued = false

function scheduleScroll() {
  if (scrollQueued) {
    return
  }
  scrollQueued = true
  requestAnimationFrame(() => {
    scrollQueued = false
    void scrollToBottom()
  })
}

async function scrollToBottom() {
  await nextTick()
  if (logBox.value) {
    logBox.value.scrollTop = logBox.value.scrollHeight
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
  try {
    streamId.value = await StartLogcat(store.current.serial, '')
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
}

// Shell
const command = ref('')
const shellOutput = ref('')
const shellBusy = ref(false)
const history = ref<string[]>([])

async function runShell() {
  if (!store.current) {
    message.warning('请先在「设备」页选择设备')
    return
  }
  const cmd = command.value.trim()
  if (!cmd) {
    return
  }

  shellBusy.value = true
  try {
    const result = await ExecShell(store.current.serial, cmd)
    const parts = [`$ ${cmd}`]
    if (result.stdout) {
      parts.push(result.stdout.trimEnd())
    }
    if (result.stderr) {
      parts.push(`[stderr]\n${result.stderr.trimEnd()}`)
    }
    parts.push(`[exit ${result.exitCode}]`)
    shellOutput.value = parts.join('\n')
    history.value.unshift(cmd)
    history.value = history.value.slice(0, 20)
  } catch (e) {
    shellOutput.value = `$ ${cmd}\n${e instanceof Error ? e.message : String(e)}`
  } finally {
    shellBusy.value = false
  }
}

let unlisten: (() => void) | null = null

onMounted(() => {
  unlisten = EventsOn('logcat:batch', (p: LogcatBatchPayload) => onBatch(p))
})

onBeforeUnmount(() => {
  // 离开页面时必须停止流并注销监听，否则事件会持续涌入导致界面卡死。
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
      <NRadioGroup v-model:value="mode" size="small">
        <NRadioButton value="logcat">实时日志</NRadioButton>
        <NRadioButton value="shell">Shell</NRadioButton>
      </NRadioGroup>
      <span class="count" v-if="mode === 'logcat'">{{ visibleLines.length }} / {{ lines.length }} 行</span>
      <div class="spacer"></div>
      <template v-if="mode === 'logcat'">
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
      </template>
    </div>

    <template v-if="mode === 'logcat'">
      <div class="filters">
        <NSelect v-model:value="level" size="small" :options="levelOptions" style="width: 140px" />
        <NInput v-model:value="keyword" size="small" clearable placeholder="关键字过滤" style="width: 220px" />
        <span class="hint">最多保留最近 1000 行</span>
      </div>

      <div ref="logBox" class="log-box">
        <div v-if="!visibleLines.length" class="log-empty">
          {{ running ? '等待日志…' : '点击「开始」读取实时日志' }}
        </div>
        <div
          v-for="(line, index) in visibleLines"
          :key="index"
          class="log-line"
          :class="`level-${line.level}`"
        >{{ line.text }}</div>
      </div>
    </template>

    <template v-else>
      <div class="shell">
        <div class="shell-input">
          <NIcon :size="16"><TerminalOutline /></NIcon>
          <span class="prompt">$</span>
          <NInput
            v-model:value="command"
            size="small"
            placeholder="输入 adb shell 命令，例如 pm list packages -3"
            @keyup.enter="runShell"
          />
          <NButton size="small" type="primary" :loading="shellBusy" @click="runShell">执行</NButton>
        </div>
        <pre class="shell-output">{{ shellOutput || '命令输出会显示在这里' }}</pre>
        <div v-if="history.length" class="history">
          <span class="hint">最近命令：</span>
          <button v-for="item in history" :key="item" class="history-chip" @click="command = item">{{ item }}</button>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.logs {
  display: flex;
  flex-direction: column;
  height: 100%;
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

.log-box {
  flex: 1;
  min-height: 420px;
  max-height: calc(100vh - 260px);
  overflow: auto;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface-inset);
}

.log-empty {
  padding: 40px 0;
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}

.log-line {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-secondary);
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

.shell {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.shell-input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.prompt {
  color: var(--primary-ink);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  font-weight: 600;
}

.shell-output {
  min-height: 380px;
  max-height: calc(100vh - 300px);
  margin: 0;
  overflow: auto;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface-inset);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
}

.history {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.history .hint {
  color: var(--text-tertiary);
  font-size: 11px;
}

.history-chip {
  padding: 3px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  cursor: pointer;
}
</style>
