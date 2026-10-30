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
  <div class="install">
    <div
      class="dropzone"
      :class="{ dragging }"
      @click="pick"
      @dragenter.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @dragover.prevent
      @drop.prevent="dragging = false"
    >
      <div class="drop-icon">
        <NIcon :size="20"><CloudUploadOutline /></NIcon>
      </div>
      <div class="drop-text">
        <div class="drop-title">{{ t('install.dropTitle') }}</div>
        <div class="drop-sub">{{ t('install.dropSub') }}</div>
      </div>
      <NButton size="small" @click.stop="pick">{{ t('install.chooseFile') }}</NButton>
    </div>

    <section v-if="pending.length" class="panel">
      <div class="panel-head">
        <h3>{{ t('install.pending') }}</h3>
        <span class="hint">{{ t('install.fileCount', { n: pending.length }) }}</span>
      </div>
      <div class="pending-list">
        <div v-for="(item, index) in pending" :key="item.path" class="pending-row">
          <div class="pending-main">
            <div class="file-name">{{ item.name }}</div>
            <div v-if="item.meta" class="file-meta">
              {{ item.meta.label || item.meta.packageName }} · {{ item.meta.versionName }}
              ({{ item.meta.versionCode }}) · {{ formatSize(item.meta.size) }}
            </div>
            <div v-else-if="item.error" class="file-meta error">{{ item.error }}</div>
          </div>
          <NButton size="tiny" quaternary @click="removeAt(index)">{{ t('install.remove') }}</NButton>
        </div>
      </div>
    </section>

    <div class="options">
      <NCheckbox v-model:checked="options.replace">{{ t('install.optReplace') }}</NCheckbox>
      <NCheckbox v-model:checked="options.allowDowngrade">{{ t('install.optDowngrade') }}</NCheckbox>
      <NCheckbox v-model:checked="options.grantPermissions">{{ t('install.optGrant') }}</NCheckbox>
      <NCheckbox v-model:checked="options.allowTest">{{ t('install.optTest') }}</NCheckbox>
    </div>

    <section class="panel queue">
      <div class="panel-head">
        <h3>{{ t('install.queue') }}</h3>
        <span class="hint">{{ t('install.queueHint') }}</span>
      </div>
      <table v-if="tasks.length" class="table">
        <thead>
          <tr>
            <th>{{ t('install.colFile') }}</th>
            <th>{{ t('install.colDevice') }}</th>
            <th>{{ t('install.colProgress') }}</th>
            <th>{{ t('install.colStatus') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in tasks" :key="task.taskId">
            <td><span class="file-name">{{ task.name }}</span></td>
            <td>{{ task.deviceLabel }}</td>
            <td>
              <div
                class="progress"
                :class="{
                  failed: task.done && !task.ok && !isCanceled(task),
                  canceled: task.done && isCanceled(task),
                }"
              >
                <i :style="{ width: `${task.percent >= 0 ? task.percent : 35}%` }"></i>
              </div>
              <div class="hint-text">{{ task.message }}</div>
            </td>
            <td>
              <div class="status-cell">
                <NTag
                  v-if="task.done"
                  size="small"
                  :type="statusType(task)"
                  :bordered="false"
                >
                  {{ statusText(task) }}
                </NTag>
                <template v-else>
                  <NTag size="small" type="info" :bordered="false">{{ t('install.running') }}</NTag>
                  <NButton size="tiny" quaternary @click="cancel(task.taskId)">{{ t('install.cancel') }}</NButton>
                </template>
              </div>
              <div v-if="task.done && !task.ok && task.advice" class="hint-text">
                {{ task.advice }}
              </div>
              <div
                v-if="task.done && !task.ok && task.code && !isCanceled(task)"
                class="hint-text code"
              >
                {{ t('install.errorCode', { code: task.code }) }}
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <NEmpty v-else class="empty" :description="t('install.emptyQueue')" />
    </section>

    <div class="actions">
      <NButton type="primary" :disabled="!canInstall" :loading="busy" @click="start">
        {{ t('install.installingTo', { name: store.current?.model || store.current?.serial || t('common.thisDevice') }) }}
      </NButton>
      <span class="actions-hint">
        {{ pending.length ? t('install.pendingCount', { n: pending.length }) : t('install.noPending') }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.install {
  --wails-drop-target: drop;
  display: flex;
  flex-direction: column;
}

.dropzone {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 22px 20px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius);
  background: var(--surface);
  cursor: pointer;
}

.dropzone.dragging {
  border-color: var(--primary);
  background: var(--primary-soft);
}

.drop-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--primary-soft);
  color: var(--primary-ink);
}

.drop-text {
  flex: 1;
}

.drop-title {
  font-size: 13px;
  font-weight: 600;
}

.drop-sub {
  margin-top: 2px;
  color: var(--text-tertiary);
  font-size: 11px;
}

.panel {
  margin-top: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.panel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
}

.panel-head h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.panel-head .hint {
  margin-left: auto;
  color: var(--text-tertiary);
  font-size: 11px;
}

.pending-list {
  display: flex;
  flex-direction: column;
}

.pending-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
}

.pending-row:last-child {
  border-bottom: none;
}

.pending-main {
  flex: 1;
  min-width: 0;
}

.file-name {
  display: block;
  max-width: 100%;
  overflow: hidden;
  font-size: 12px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  margin-top: 2px;
  overflow: hidden;
  color: var(--text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta.error {
  color: var(--danger);
}

.options {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 16px;
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
}

.table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.table th {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  color: var(--text-tertiary);
  font-size: 10px;
  font-weight: 500;
  text-align: left;
  text-transform: uppercase;
}

.table th:nth-child(1),
.table td:nth-child(1) {
  width: 210px;
}

.table th:nth-child(2),
.table td:nth-child(2) {
  width: 130px;
}

.table th:nth-child(3),
.table td:nth-child(3) {
  width: 220px;
}

.table td {
  padding: 11px 12px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  vertical-align: top;
}

.table tr:last-child td {
  border-bottom: none;
}

.table th:last-child,
.table td:last-child {
  padding-right: 18px;
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress {
  width: 180px;
  height: 6px;
  border-radius: 3px;
  background: var(--border);
  overflow: hidden;
}

.progress > i {
  display: block;
  height: 100%;
  border-radius: 3px;
  background: var(--primary);
  transition: width 0.3s ease;
}

.progress.failed > i {
  background: var(--danger);
}

.progress.canceled > i {
  background: var(--text-tertiary);
}

.hint-text.code {
  color: var(--text-tertiary);
  font-size: 10px;
}

.hint-text {
  margin-top: 4px;
  color: var(--text-tertiary);
  font-size: 11px;
  line-height: 1.5;
  word-break: break-all;
}

.empty {
  padding: 28px 0;
}

.actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
}

.actions-hint {
  color: var(--text-tertiary);
  font-size: 12px;
}
</style>
