<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NIcon, NInput, useMessage } from 'naive-ui'
import { CameraOutline, CloudDownloadOutline, CloudUploadOutline } from '@vicons/ionicons5'
import { DownloadDir, PickDirectory, PickFile, PullFile, PushFile, Screenshot } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import DevicePathPicker from '../components/DevicePathPicker.vue'
import { useDevicesStore } from '../stores/devices'
import { t } from '../i18n'

interface FileProgressPayload {
  stage: string
  percent: number
  message: string
}

interface DropPayload {
  x: number
  y: number
  paths: string[]
}

const store = useDevicesStore()
const message = useMessage()

const shot = ref<model.ScreenshotResult | null>(null)
const shooting = ref(false)

const localFiles = ref<string[]>([])
const pushRemote = ref('/sdcard/Download/')
const pushing = ref(false)

const remoteFiles = ref<string[]>([])
const pullLocalDir = ref('')
const pulling = ref(false)

const transfer = ref({ active: false, label: '', percent: 0 })

const pickerShow = ref(false)
const pickerMode = ref<'file' | 'dir'>('dir')
const pickerTarget = ref<'push' | 'pull'>('push')
const pickerMulti = ref(false)

const canPush = computed(
  () => !!store.current && localFiles.value.length > 0 && !!pushRemote.value.trim() && !pushing.value,
)
const canPull = computed(
  () => !!store.current && remoteFiles.value.length > 0 && !!pullLocalDir.value && !pulling.value,
)

function baseName(path: string) {
  const parts = path.split('/')
  return parts[parts.length - 1] || path
}

function addLocalFiles(paths: string[]) {
  for (const path of paths) {
    if (path && !localFiles.value.includes(path)) {
      localFiles.value.push(path)
    }
  }
}

async function chooseLocal() {
  try {
    const path = await PickFile()
    if (path) {
      addLocalFiles([path])
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function chooseDir() {
  try {
    const dir = await PickDirectory()
    if (dir) {
      pullLocalDir.value = dir
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

function openPicker(mode: 'file' | 'dir', target: 'push' | 'pull', multi = false) {
  if (!store.current) {
    message.warning(t('common.selectDevice'))
    return
  }
  pickerMode.value = mode
  pickerTarget.value = target
  pickerMulti.value = multi
  pickerShow.value = true
}

function onPickerSelect(path: string) {
  if (pickerTarget.value === 'push') {
    pushRemote.value = path.endsWith('/') ? path : `${path}/`
    return
  }
  remoteFiles.value = [path]
}

function onPickerSelectMany(paths: string[]) {
  if (pickerTarget.value !== 'pull') {
    return
  }
  remoteFiles.value = [...new Set([...remoteFiles.value, ...paths])]
}

async function doPush() {
  if (!store.current) {
    message.warning(t('common.selectDevice'))
    return
  }
  if (!localFiles.value.length || !pushRemote.value.trim()) {
    message.warning(t('files.needLocalAndTarget'))
    return
  }

  pushing.value = true
  const total = localFiles.value.length
  try {
    for (let i = 0; i < total; i += 1) {
      transfer.value = { active: true, label: t('files.pushing', { i: i + 1, total }), percent: 0 }
      await PushFile(store.current.serial, localFiles.value[i], pushRemote.value.trim())
    }
    message.success(t('files.pushed', { n: total }))
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    pushing.value = false
    transfer.value.active = false
  }
}

async function doPull() {
  if (!store.current) {
    message.warning(t('common.selectDevice'))
    return
  }
  if (!remoteFiles.value.length || !pullLocalDir.value) {
    message.warning(t('files.needRemoteAndDir'))
    return
  }

  pulling.value = true
  const total = remoteFiles.value.length
  try {
    for (let i = 0; i < total; i += 1) {
      transfer.value = { active: true, label: t('files.pulling', { i: i + 1, total }), percent: 0 }
      await PullFile(store.current.serial, remoteFiles.value[i], pullLocalDir.value)
    }
    message.success(t('files.pulled', { n: total }))
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    pulling.value = false
    transfer.value.active = false
  }
}

async function takeShot() {
  if (!store.current) {
    message.warning(t('common.selectDevice'))
    return
  }
  shooting.value = true
  try {
    shot.value = await Screenshot(store.current.serial)
    message.success(t('files.shotSaved'))
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    shooting.value = false
  }
}

function onProgress(p: FileProgressPayload) {
  transfer.value.active = true
  transfer.value.label = p.message
  transfer.value.percent = Math.max(transfer.value.percent, p.percent)
}

onMounted(async () => {
  try {
    const dir = await DownloadDir()
    if (dir) {
      pullLocalDir.value = dir
    }
  } catch {
    // 忽略，保留手动选择
  }
  EventsOn('file:progress', (p: FileProgressPayload) => onProgress(p))
  EventsOn('files:dropped', (payload: DropPayload | string[]) => {
    const paths = Array.isArray(payload) ? payload : payload.paths
    addLocalFiles(paths ?? [])
  })
})
</script>

<template>
  <div class="files">
    <NAlert v-if="!store.current" type="warning" class="alert">
      {{ t('files.selectDevice') }}
    </NAlert>

    <div v-if="transfer.active" class="transfer-bar">
      <span class="transfer-label">{{ transfer.label }}</span>
      <div class="transfer-progress"><i :style="{ width: `${transfer.percent}%` }"></i></div>
      <span class="transfer-percent">{{ transfer.percent }}%</span>
    </div>

    <div class="transfer-grid">
      <section class="panel side">
        <div class="panel-head">
          <NIcon :size="16"><CloudUploadOutline /></NIcon>
          <h3>{{ t('files.localTitle') }}</h3>
          <span class="hint">{{ t('files.dragHint') }}</span>
          <NButton v-if="localFiles.length" size="small" quaternary @click="localFiles = []">{{ t('files.clear') }}</NButton>
          <NButton size="small" @click="chooseLocal">{{ t('files.select') }}</NButton>
        </div>
        <div class="panel-body">
          <div class="drop-hint">{{ t('files.dropHintLocal') }}</div>
          <ul class="file-list">
            <li v-for="(file, index) in localFiles" :key="file">
              <span class="file-name" :title="file">{{ baseName(file) }}</span>
              <button class="remove" :title="t('files.remove')" @click="localFiles.splice(index, 1)">×</button>
            </li>
            <li v-if="!localFiles.length" class="empty">{{ t('files.emptyList') }}</li>
          </ul>
          <div class="row-line">
            <span class="row-label">{{ t('files.targetPath') }}</span>
            <NInput v-model:value="pushRemote" size="small" :placeholder="t('files.targetPathHint')" />
            <NButton size="small" @click="openPicker('dir', 'push')">{{ t('files.browse') }}</NButton>
            <NButton size="small" type="primary" :loading="pushing" :disabled="!canPush" @click="doPush">
              {{ localFiles.length > 1 ? t('files.pushN', { n: localFiles.length }) : t('files.push') }}
            </NButton>
          </div>
        </div>
      </section>

      <section class="panel side">
        <div class="panel-head">
          <NIcon :size="16"><CloudDownloadOutline /></NIcon>
          <h3>{{ t('files.remoteTitle') }}</h3>
          <span class="hint">{{ t('files.multiHint') }}</span>
          <NButton v-if="remoteFiles.length" size="small" quaternary @click="remoteFiles = []">{{ t('files.clear') }}</NButton>
          <NButton size="small" @click="openPicker('file', 'pull', true)">{{ t('files.browseDevice') }}</NButton>
        </div>
        <div class="panel-body">
          <div class="drop-hint">{{ t('files.dropHintRemote') }}</div>
          <ul class="file-list">
            <li v-for="(file, index) in remoteFiles" :key="file">
              <span class="file-name" :title="file">{{ baseName(file) }}</span>
              <button class="remove" :title="t('files.remove')" @click="remoteFiles.splice(index, 1)">×</button>
            </li>
            <li v-if="!remoteFiles.length" class="empty">{{ t('files.emptyList') }}</li>
          </ul>
          <div class="row-line">
            <span class="row-label">{{ t('files.saveTo') }}</span>
            <NInput :value="pullLocalDir" size="small" readonly :placeholder="t('files.chooseDir')" />
            <NButton size="small" @click="chooseDir">{{ t('files.chooseDir') }}</NButton>
            <NButton size="small" type="primary" :loading="pulling" :disabled="!canPull" @click="doPull">
              {{ remoteFiles.length > 1 ? t('files.pullN', { n: remoteFiles.length }) : t('files.pull') }}
            </NButton>
          </div>
        </div>
      </section>
    </div>

    <section class="panel">
      <div class="panel-head">
        <NIcon :size="16"><CameraOutline /></NIcon>
        <h3>{{ t('files.screenshot') }}</h3>
        <span class="hint">{{ t('files.screenshotHint') }}</span>
        <NButton size="small" type="primary" :loading="shooting" :disabled="!store.current" @click="takeShot">
          {{ t('files.takeShot') }}
        </NButton>
      </div>
      <div class="panel-body">
        <div v-if="shot" class="shot">
          <img :src="shot.dataUrl" class="shot-image" :alt="t('files.shotPreview')" />
          <div class="shot-path">{{ shot.path }}</div>
        </div>
        <div v-else class="shot-empty">{{ t('files.noShot') }}</div>
      </div>
    </section>

    <DevicePathPicker
      v-model:show="pickerShow"
      :serial="store.current?.serial || ''"
      :mode="pickerMode"
      :multi="pickerMulti"
      @select="onPickerSelect"
      @select-many="onPickerSelectMany"
    />
  </div>
</template>

<style scoped>
.files {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.alert {
  margin-bottom: 0;
}

.transfer-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--primary-border);
  border-radius: var(--radius-sm);
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 12px;
}

.transfer-label {
  width: 110px;
  white-space: nowrap;
}

.transfer-progress {
  flex: 1;
  height: 5px;
  overflow: hidden;
  border-radius: 3px;
  background: var(--border);
}

.transfer-progress > i {
  display: block;
  height: 100%;
  background: var(--primary);
  transition: width 0.2s ease;
}

.transfer-percent {
  width: 36px;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.transfer-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.panel {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.panel.side {
  display: flex;
  flex-direction: column;
}

.panel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.panel-head h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.panel-head .hint {
  margin-left: auto;
  margin-right: 8px;
  color: var(--text-tertiary);
  font-size: 11px;
}

.panel-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
}

.drop-hint {
  padding: 10px 12px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--text-tertiary);
  font-size: 11px;
  text-align: center;
}

.file-list {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-height: 120px;
  max-height: 260px;
  margin: 0;
  padding: 6px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-inset);
  list-style: none;
}

.file-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border-radius: 6px;
  font-size: 12px;
}

.file-list li:hover {
  background: var(--surface);
}

.file-list .file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-list .empty {
  justify-content: center;
  padding: 24px 0;
  color: var(--text-tertiary);
  font-size: 12px;
}

.file-list .empty:hover {
  background: transparent;
}

.remove {
  flex: 0 0 auto;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--text-tertiary);
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
}

.remove:hover {
  background: var(--danger-soft);
  color: var(--danger);
}

.row-line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.row-label {
  flex: 0 0 60px;
  color: var(--text-secondary);
  font-size: 12px;
}

.shot {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.shot-image {
  max-width: 100%;
  max-height: 420px;
  object-fit: contain;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-inset);
}

.shot-path {
  color: var(--text-tertiary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  word-break: break-all;
}

.shot-empty {
  padding: 24px 0;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}
</style>
