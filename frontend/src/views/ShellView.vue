<script setup lang="ts">
import { nextTick, ref , computed } from 'vue'
import { NButton, NIcon, NInput, useMessage } from 'naive-ui'
import { CopyOutline, PlayOutline, TrashOutline } from '@vicons/ionicons5'
import { CopyText, ExecShell } from '../../wailsjs/go/main/App'
import { useDevicesStore } from '../stores/devices'
import { t } from '../i18n'

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

const quickCommands = computed(() => [
  { label: t('shell.quickModel'), command: 'getprop ro.product.model' },
  { label: t('shell.quickApps'), command: 'pm list packages -3' },
  { label: t('shell.quickBattery'), command: 'dumpsys battery' },
  { label: t('shell.quickSize'), command: 'wm size' },
  { label: t('shell.quickTop'), command: 'top -n 1 -b | head -20' },
])

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
    message.warning(t('common.selectDevice'))
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
  message.success(t('shell.copied'))
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div class="mb-2.5 flex items-center gap-2">
      <span class="text-xs text-fg2">
        {{ store.current?.model || store.current?.serial || t('common.unknown') }}
      </span>
      <div class="flex-1"></div>
      <NButton size="small" quaternary :disabled="!records.length" @click="copyAll">
        <template #icon><NIcon><CopyOutline /></NIcon></template>
        {{ t('shell.copyAll') }}
      </NButton>
      <NButton size="small" quaternary :disabled="!records.length" @click="clear">
        <template #icon><NIcon><TrashOutline /></NIcon></template>
        {{ t('shell.clear') }}
      </NButton>
    </div>

    <div class="mb-2.5 flex flex-wrap items-center gap-2">
      <span class="text-[11px] text-fg3">{{ t('shell.quick') }}</span>
      <button
        v-for="item in quickCommands"
        :key="item.command"
        class="cursor-pointer rounded-full border border-line bg-surface px-2.5 py-1 text-[11px] text-fg2 hover:border-brand-line hover:text-brand-ink"
        @click="useQuick(item.command)"
      >
        {{ item.label }}
      </button>
    </div>

    <div ref="outputBox" class="min-h-0 flex-1 overflow-y-auto rounded border border-line bg-inset p-3">
      <div v-if="!records.length" class="py-15 text-center text-xs text-fg3">
        {{ t('shell.empty') }}
      </div>
      <div v-for="record in records" :key="record.id" class="mb-3.5 last:mb-0">
        <div class="flex items-center gap-2 rounded-[6px] border border-line bg-surface px-2.5 py-1.75 font-mono text-xs">
          <span class="font-700 text-brand-ink">$</span>
          <span class="break-all font-600 text-fg">{{ record.command }}</span>
          <span class="ml-auto flex gap-2.5 whitespace-nowrap text-[10px] text-fg3">
            <span
              class="rounded-[4px] px-1.5"
              :class="record.exitCode === 0 ? 'bg-[var(--success-soft)] text-ok' : 'bg-err-soft text-err'"
            >
              exit {{ record.exitCode }}
            </span>
            <span>{{ record.durationMs }}ms</span>
            <span>{{ record.at }}</span>
          </span>
        </div>
        <pre
          v-if="record.stdout"
          class="mb-0 ml-3.5 mt-1.25 overflow-auto whitespace-pre-wrap break-all rounded-r-[6px] border-l-2 border-line-strong bg-surface px-2.5 py-2 font-mono text-[11px] leading-[1.7] text-fg2"
        >{{ record.stdout }}</pre>
        <pre
          v-if="record.stderr"
          class="mb-0 ml-3.5 mt-1.25 overflow-auto whitespace-pre-wrap break-all rounded-r-[6px] border-l-2 border-err bg-surface px-2.5 py-2 font-mono text-[11px] leading-[1.7] text-err"
        >{{ record.stderr }}</pre>
      </div>
    </div>

    <div class="mt-2.5 flex items-center gap-2">
      <span class="font-mono text-sm font-700 text-brand-ink">$</span>
      <NInput v-model:value="command" size="small" :placeholder="t('shell.placeholder')" @keyup.enter="run" />
      <NButton size="small" type="primary" :loading="busy" :disabled="!store.current" @click="run">
        <template #icon><NIcon><PlayOutline /></NIcon></template>
        {{ t('shell.run') }}
      </NButton>
    </div>
  </div>
</template>
