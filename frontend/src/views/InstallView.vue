<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NCheckbox, NEmpty, NIcon, NTag, useMessage } from 'naive-ui'
import { CloudUploadOutline } from '@vicons/ionicons5'
import {
  CancelInstall,
  InspectApk,
  PickApkFile,
  StartInstall,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { model } from '../../wailsjs/go/models'
import { useDevicesStore } from '../stores/devices'
import { t } from '../i18n'

interface PendingApk {
  path: string
  name: string
  meta?: model.ApkMeta | null
  error?: string
}

// 事件负载类型不会由 Wails 生成，这里按 Go 端结构手写。
interface TaskProgressPayload {
  taskId: string
  stage: string
  percent: number
  message: string
}

interface InstallResultPayload {
  taskId: string
  ok: boolean
  code: string
  message: string
  advice: string
  raw: string
  durationMs: number
}

interface TaskRow {
  taskId: string
  name: string
  deviceLabel: string
  stage: string
  percent: number
  message: string
  done: boolean
  ok: boolean
  code: string
  advice: string
}

const store = useDevicesStore()
const message = useMessage()

const pending = ref<PendingApk[]>([])
const tasks = ref<TaskRow[]>([])
const busy = ref(false)
const dragging = ref(false)
const options = ref({
  replace: true,
  allowDowngrade: false,
  grantPermissions: false,
  allowTest: false,
})

const canInstall = computed(() => pending.value.length > 0 && !!store.current)

function formatSize(bytes: number) {
  if (bytes >= 1024 * 1024 * 1024) {
    return `${(bytes / 1024 ** 3).toFixed(1)} GB`
  }
  return `${(bytes / 1024 ** 2).toFixed(1)} MB`
}

function fileName(path: string) {
  return path.split('/').pop() || path
}

async function addFiles(paths: string[]) {
  for (const path of paths) {
    if (!path.toLowerCase().endsWith('.apk')) {
      continue
    }
    const item: PendingApk = { path, name: fileName(path) }
    try {
      item.meta = await InspectApk(path)
    } catch (e) {
      item.error = e instanceof Error ? e.message : String(e)
    }
    pending.value.push(item)
  }
}

async function pick() {
  try {
    const path = await PickApkFile()
    if (path) {
      await addFiles([path])
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

function removeAt(index: number) {
  pending.value.splice(index, 1)
}

async function cancel(taskId: string) {
  try {
    await CancelInstall(taskId)
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

function deviceLabel() {
  if (!store.current) {
    return '—'
  }
  return store.current.model || store.current.serial
}

async function start() {
  if (!store.current) {
    message.warning(t('install.selectDeviceFirst'))
    return
  }

  busy.value = true
  try {
    for (const item of pending.value) {
      const taskId = await StartInstall(
        model.InstallRequest.createFrom({
          serial: store.current.serial,
          paths: [item.path],
          options: options.value,
        }),
      )
      tasks.value.unshift({
        taskId,
        name: item.name,
        deviceLabel: deviceLabel(),
        stage: 'queued',
        percent: 0,
        message: t('install.queued'),
        done: false,
        ok: false,
        code: '',
        advice: '',
      })
    }
    pending.value = []
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    busy.value = false
  }
}

function findTask(taskId: string) {
  return tasks.value.find((item) => item.taskId === taskId)
}

function onProgress(p: TaskProgressPayload) {
  const task = findTask(p.taskId)
  if (!task) {
    return
  }
  task.stage = p.stage
  // 阶段事件与传输回调可能交错，单调递增避免进度条视觉回退。
  if (p.percent >= 0) {
    task.percent = Math.max(task.percent, p.percent)
  }
  task.message = p.message
}

function onDone(r: InstallResultPayload) {
  const task = findTask(r.taskId)
  if (!task) {
    return
  }
  task.done = true
  task.ok = r.ok
  task.code = r.code
  task.advice = r.advice

  if (r.ok) {
    task.percent = 100
    task.message = t('install.successWithDuration', { s: (r.durationMs / 1000).toFixed(1) })
    message.success(`${task.name} · ${t('install.statusInstalled')}`)
  } else if (isCanceled(task)) {
    task.message = t('install.cancelled')
    message.info(`${task.name} · ${t('install.statusCancelled')}`)
  } else {
    task.message = t('install.failed')
    message.error(`${task.name} · ${t('install.statusFailed')}`)
  }
}

function isCanceled(task: TaskRow) {
  return task.code === 'CANCELED'
}

function statusText(task: TaskRow) {
  if (task.ok) {
    return t('install.statusInstalled')
  }
  if (isCanceled(task)) {
    return t('install.statusCancelled')
  }
  return t('install.statusFailed')
}

function statusType(task: TaskRow): 'success' | 'default' | 'error' {
  if (task.ok) {
    return 'success'
  }
  if (isCanceled(task)) {
    return 'default'
  }
  return 'error'
}

onMounted(() => {
  EventsOn('files:dropped', (paths: string[]) => {
    void addFiles(paths)
  })
  EventsOn('task:progress', (p: TaskProgressPayload) => onProgress(p))
  EventsOn('task:done', (r: InstallResultPayload) => onDone(r))
})
</script>

<template>
  <div class="flex flex-col">
    <div
      class="flex cursor-pointer items-center gap-3.5 rounded border border-dashed bg-surface px-5 py-[22px] [--wails-drop-target:drop]"
      :class="dragging ? 'border-brand bg-brand-soft' : 'border-line-strong'"
      @click="pick"
      @dragenter.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @dragover.prevent
      @drop.prevent="dragging = false"
    >
      <div class="flex h-10 w-10 items-center justify-center rounded-[10px] bg-brand-soft text-brand-ink">
        <NIcon :size="20"><CloudUploadOutline /></NIcon>
      </div>
      <div class="flex-1">
        <div class="text-[13px] font-600">{{ t('install.dropTitle') }}</div>
        <div class="mt-0.5 text-[11px] text-fg3">{{ t('install.dropSub') }}</div>
      </div>
      <NButton size="small" @click.stop="pick">{{ t('install.chooseFile') }}</NButton>
    </div>

    <section v-if="pending.length" class="mt-4 rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('install.pending') }}</h3>
        <span class="ml-auto text-[11px] text-fg3">{{ t('install.fileCount', { n: pending.length }) }}</span>
      </div>
      <div class="flex flex-col">
        <div
          v-for="(item, index) in pending"
          :key="item.path"
          class="flex items-center gap-3 border-b border-line px-4 py-2.5 last:border-b-0"
        >
          <div class="min-w-0 flex-1">
            <div class="overflow-hidden text-ellipsis whitespace-nowrap text-xs font-500">{{ item.name }}</div>
            <div v-if="item.meta" class="mt-0.5 overflow-hidden text-ellipsis whitespace-nowrap text-[11px] text-fg3">
              {{ item.meta.label || item.meta.packageName }} · {{ item.meta.versionName }}
              ({{ item.meta.versionCode }}) · {{ formatSize(item.meta.size) }}
            </div>
            <div v-else-if="item.error" class="mt-0.5 overflow-hidden text-ellipsis whitespace-nowrap text-[11px] text-err">
              {{ item.error }}
            </div>
          </div>
          <NButton size="tiny" quaternary @click="removeAt(index)">{{ t('install.remove') }}</NButton>
        </div>
      </div>
    </section>

    <div class="mt-4 flex flex-wrap gap-4 rounded-sm border border-line bg-surface px-4 py-3">
      <NCheckbox v-model:checked="options.replace">{{ t('install.optReplace') }}</NCheckbox>
      <NCheckbox v-model:checked="options.allowDowngrade">{{ t('install.optDowngrade') }}</NCheckbox>
      <NCheckbox v-model:checked="options.grantPermissions">{{ t('install.optGrant') }}</NCheckbox>
      <NCheckbox v-model:checked="options.allowTest">{{ t('install.optTest') }}</NCheckbox>
    </div>

    <section class="mt-4 rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('install.queue') }}</h3>
        <span class="ml-auto text-[11px] text-fg3">{{ t('install.queueHint') }}</span>
      </div>
      <table v-if="tasks.length" class="w-full table-fixed border-collapse">
        <thead>
          <tr>
            <th class="w-[210px] border-b border-line px-3 py-2 text-left text-[10px] font-500 uppercase text-fg3">{{ t('install.colFile') }}</th>
            <th class="w-[130px] border-b border-line px-3 py-2 text-left text-[10px] font-500 uppercase text-fg3">{{ t('install.colDevice') }}</th>
            <th class="w-[220px] border-b border-line px-3 py-2 text-left text-[10px] font-500 uppercase text-fg3">{{ t('install.colProgress') }}</th>
            <th class="border-b border-line px-3 py-2 pr-4.5 text-left text-[10px] font-500 uppercase text-fg3">{{ t('install.colStatus') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in tasks" :key="task.taskId">
            <td class="border-b border-line px-3 py-2.75 align-top text-xs last:border-b-0">
              <span class="block overflow-hidden text-ellipsis whitespace-nowrap text-xs font-500">{{ task.name }}</span>
            </td>
            <td class="border-b border-line px-3 py-2.75 align-top text-xs">{{ task.deviceLabel }}</td>
            <td class="border-b border-line px-3 py-2.75 align-top text-xs">
              <div
                class="h-1.5 w-[180px] overflow-hidden rounded-[3px] bg-line"
              >
                <i
                  class="block h-full rounded-[3px] transition-[width] duration-300"
                  :class="
                    task.done && !task.ok && !isCanceled(task)
                      ? 'bg-err'
                      : task.done && isCanceled(task)
                        ? 'bg-fg3'
                        : 'bg-brand'
                  "
                  :style="{ width: `${task.percent >= 0 ? task.percent : 35}%` }"
                ></i>
              </div>
              <div class="mt-1 break-all text-[11px] leading-normal text-fg3">{{ task.message }}</div>
            </td>
            <td class="border-b border-line px-3 py-2.75 pr-4.5 align-top text-xs last:border-b-0">
              <div class="flex items-center gap-2">
                <NTag v-if="task.done" size="small" :type="statusType(task)" :bordered="false">
                  {{ statusText(task) }}
                </NTag>
                <template v-else>
                  <NTag size="small" type="info" :bordered="false">{{ t('install.running') }}</NTag>
                  <NButton size="tiny" quaternary @click="cancel(task.taskId)">{{ t('install.cancel') }}</NButton>
                </template>
              </div>
              <div v-if="task.done && !task.ok && task.advice" class="mt-1 break-all text-[11px] leading-normal text-fg3">
                {{ task.advice }}
              </div>
              <div
                v-if="task.done && !task.ok && task.code && !isCanceled(task)"
                class="mt-1 break-all text-[10px] leading-normal text-fg3"
              >
                {{ t('install.errorCode', { code: task.code }) }}
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <NEmpty v-else class="py-7" :description="t('install.emptyQueue')" />
    </section>

    <div class="mt-4 flex items-center gap-3">
      <NButton type="primary" :disabled="!canInstall" :loading="busy" @click="start">
        {{ t('install.installingTo', { name: store.current?.model || store.current?.serial || t('common.thisDevice') }) }}
      </NButton>
      <span class="text-xs text-fg3">
        {{ pending.length ? t('install.pendingCount', { n: pending.length }) : t('install.noPending') }}
      </span>
    </div>
  </div>
</template>
