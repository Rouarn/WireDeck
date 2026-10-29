<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { NButton, NIcon, NInput, useMessage } from 'naive-ui'
import { CopyOutline, PlayOutline, TrashOutline } from '@vicons/ionicons5'
import { CopyText, ExecShell } from '../../wailsjs/go/main/App'
import { useDevicesStore } from '../stores/devices'

interface ShellRecord {
  id: number
  command: string
  stdout: string
  stderr: string
  exitCode: number
  durationMs: number
  at: string
}

const store = useDevicesStore()
const message = useMessage()

const command = ref('')
const busy = ref(false)
const records = ref<ShellRecord[]>([])
const outputBox = ref<HTMLElement | null>(null)
const seq = ref(0)

const quickCommands = [
  { label: '设备型号', command: 'getprop ro.product.model' },
  { label: '第三方应用', command: 'pm list packages -3' },
  { label: '电池状态', command: 'dumpsys battery' },
  { label: '屏幕分辨率', command: 'wm size' },
  { label: '进程占用', command: 'top -n 1 -b | head -20' },
]

function formatTime() {
  return new Date().toLocaleTimeString('zh-CN', { hour12: false })
}

function scrollToBottom() {
  const el = outputBox.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}

async function run() {
  if (!store.current) {
    message.warning('请先在「设备」页选择设备')
    return
  }
  const cmd = command.value.trim()
  if (!cmd) {
    return
  }

  busy.value = true
  const started = Date.now()

  try {
    const result = await ExecShell(store.current.serial, cmd)
    records.value.push({
      id: ++seq.value,
      command: cmd,
      stdout: result.stdout.trimEnd(),
      stderr: result.stderr.trimEnd(),
      exitCode: result.exitCode,
      durationMs: Date.now() - started,
      at: formatTime(),
    })
    command.value = ''
  } catch (e) {
    records.value.push({
      id: ++seq.value,
      command: cmd,
      stdout: '',
      stderr: e instanceof Error ? e.message : String(e),
      exitCode: -1,
      durationMs: Date.now() - started,
      at: formatTime(),
    })
  } finally {
    busy.value = false
    await nextTick()
    scrollToBottom()
  }
}

function clear() {
  records.value = []
}

function useQuick(cmd: string) {
  command.value = cmd
}

async function copyAll() {
  if (!records.value.length) {
    return
  }
  const text = records.value
    .map((record) => {
      const parts = [`$ ${record.command}`]
      if (record.stdout) {
        parts.push(record.stdout)
      }
      if (record.stderr) {
        parts.push(`[stderr]\n${record.stderr}`)
      }
      parts.push(`[exit ${record.exitCode}]`)
      return parts.join('\n')
    })
    .join('\n\n')

  await CopyText(text)
  message.success('输出已复制')
}
</script>

<template>
  <div class="shell">
    <div class="toolbar">
      <span class="device">
        {{ store.current?.model || store.current?.serial || '未选择设备' }}
      </span>
      <div class="spacer"></div>
      <NButton size="small" quaternary :disabled="!records.length" @click="copyAll">
        <template #icon><NIcon><CopyOutline /></NIcon></template>
        复制全部
      </NButton>
      <NButton size="small" quaternary :disabled="!records.length" @click="clear">
        <template #icon><NIcon><TrashOutline /></NIcon></template>
        清空
      </NButton>
    </div>

    <div class="quick">
      <span class="quick-label">快捷命令</span>
      <button
        v-for="item in quickCommands"
        :key="item.command"
        class="quick-chip"
        @click="useQuick(item.command)"
      >
        {{ item.label }}
      </button>
    </div>

    <div ref="outputBox" class="output">
      <div v-if="!records.length" class="output-empty">
        输入命令开始，例如 <code>getprop ro.product.model</code>
      </div>
      <div v-for="record in records" :key="record.id" class="record">
        <div class="record-head">
          <span class="prompt">$</span>
          <span class="record-command">{{ record.command }}</span>
          <span class="record-meta">
            <span class="exit" :class="record.exitCode === 0 ? 'ok' : 'bad'">exit {{ record.exitCode }}</span>
            <span>{{ record.durationMs }}ms</span>
            <span>{{ record.at }}</span>
          </span>
        </div>
        <pre v-if="record.stdout" class="record-out">{{ record.stdout }}</pre>
        <pre v-if="record.stderr" class="record-err">{{ record.stderr }}</pre>
      </div>
    </div>

    <div class="input-bar">
      <span class="prompt">$</span>
      <NInput
        v-model:value="command"
        size="small"
        placeholder="输入 adb shell 命令，按 Enter 执行"
        @keyup.enter="run"
      />
      <NButton size="small" type="primary" :loading="busy" :disabled="!store.current" @click="run">
        <template #icon><NIcon><PlayOutline /></NIcon></template>
        执行
      </NButton>
    </div>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.device {
  color: var(--text-secondary);
  font-size: 12px;
}

.spacer {
  flex: 1;
}

.quick {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.quick-label {
  color: var(--text-tertiary);
  font-size: 11px;
}

.quick-chip {
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-secondary);
  font-size: 11px;
  cursor: pointer;
}

.quick-chip:hover {
  border-color: var(--primary-border);
  color: var(--primary-ink);
}

.output {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface-inset);
}

.output-empty {
  padding: 60px 0;
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}

.output-empty code {
  padding: 1px 6px;
  border-radius: 5px;
  background: var(--surface);
  color: var(--text-secondary);
}

.record {
  margin-bottom: 14px;
}

.record:last-child {
  margin-bottom: 0;
}

/* 命令行做成独立卡片，与输出拉开层次 */
.record-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}

.prompt {
  color: var(--primary-ink);
  font-weight: 700;
}

.record-command {
  color: var(--text);
  font-weight: 600;
  word-break: break-all;
}

.record-meta {
  display: flex;
  gap: 10px;
  margin-left: auto;
  color: var(--text-tertiary);
  font-size: 10px;
  white-space: nowrap;
}

.exit {
  padding: 0 6px;
  border-radius: 4px;
}

.exit.ok {
  background: var(--success-soft);
  color: var(--success);
}

.exit.bad {
  background: var(--danger-soft);
  color: var(--danger);
}

/* 输出缩进 + 左侧竖线，视觉上从属于上方的命令 */
.record-out,
.record-err {
  margin: 5px 0 0 14px;
  padding: 8px 10px;
  overflow: auto;
  border-left: 2px solid var(--border-strong);
  border-radius: 0 6px 6px 0;
  background: var(--surface);
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
}

.record-err {
  border-left-color: var(--danger);
  color: var(--danger);
}

.input-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
}

.input-bar .prompt {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 14px;
}
</style>
