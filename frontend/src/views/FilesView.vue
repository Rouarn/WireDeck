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
  <div class="flex flex-col gap-4">
    <NAlert v-if="!store.current" type="warning" class="mb-0">
      {{ t('files.selectDevice') }}
    </NAlert>

    <div
      v-if="transfer.active"
      class="flex items-center gap-2.5 rounded-sm border border-brand-line bg-brand-soft px-3 py-2 text-xs text-brand-ink"
    >
      <span class="w-[110px] whitespace-nowrap">{{ transfer.label }}</span>
      <div class="h-[5px] flex-1 overflow-hidden rounded-[3px] bg-line">
        <i class="block h-full bg-brand transition-[width] duration-200" :style="{ width: `${transfer.percent}%` }"></i>
      </div>
      <span class="w-9 text-right tabular-nums">{{ transfer.percent }}%</span>
    </div>

    <div class="grid grid-cols-2 gap-4">
      <section class="flex flex-col rounded border border-line bg-surface shadow-sm">
        <div class="flex items-center gap-2 border-b border-line px-4 py-3">
          <NIcon :size="16"><CloudUploadOutline /></NIcon>
          <h3 class="m-0 text-[13px] font-600">{{ t('files.localTitle') }}</h3>
          <span class="ml-auto mr-2 text-[11px] text-fg3">{{ t('files.dragHint') }}</span>
          <NButton v-if="localFiles.length" size="small" quaternary @click="localFiles = []">{{ t('files.clear') }}</NButton>
          <NButton size="small" @click="chooseLocal">{{ t('files.select') }}</NButton>
        </div>
        <div class="flex flex-1 flex-col gap-2.5 p-4">
          <div class="rounded-sm border border-dashed border-line-strong px-3 py-2.5 text-center text-[11px] text-fg3">
            {{ t('files.dropHintLocal') }}
          </div>
          <ul class="m-0 flex max-h-[260px] min-h-[120px] flex-1 list-none flex-col gap-1 overflow-y-auto rounded-sm border border-line bg-inset p-1.5">
            <li v-for="(file, index) in localFiles" :key="file" class="flex items-center gap-2 rounded-[6px] px-2 py-1.25 text-xs hover:bg-surface">
              <span class="flex-1 overflow-hidden text-ellipsis whitespace-nowrap" :title="file">{{ baseName(file) }}</span>
              <button
                class="h-[18px] w-[18px] shrink-0 cursor-pointer rounded-full border-0 bg-transparent text-sm leading-none text-fg3 hover:bg-err-soft hover:text-err"
                :title="t('files.remove')"
                @click="localFiles.splice(index, 1)"
              >
                ×
              </button>
            </li>
            <li v-if="!localFiles.length" class="flex justify-center rounded-[6px] py-6 text-center text-xs text-fg3">
              {{ t('files.emptyList') }}
            </li>
          </ul>
          <div class="flex items-center gap-2">
            <span class="min-w-[60px] shrink-0 whitespace-nowrap text-xs text-fg2">{{ t('files.targetPath') }}</span>
            <NInput v-model:value="pushRemote" size="small" :placeholder="t('files.targetPathHint')" />
            <NButton size="small" @click="openPicker('dir', 'push')">{{ t('files.browse') }}</NButton>
            <NButton size="small" type="primary" :loading="pushing" :disabled="!canPush" @click="doPush">
              {{ localFiles.length > 1 ? t('files.pushN', { n: localFiles.length }) : t('files.push') }}
            </NButton>
          </div>
        </div>
      </section>

      <section class="flex flex-col rounded border border-line bg-surface shadow-sm">
        <div class="flex items-center gap-2 border-b border-line px-4 py-3">
          <NIcon :size="16"><CloudDownloadOutline /></NIcon>
          <h3 class="m-0 text-[13px] font-600">{{ t('files.remoteTitle') }}</h3>
          <span class="ml-auto mr-2 text-[11px] text-fg3">{{ t('files.multiHint') }}</span>
          <NButton v-if="remoteFiles.length" size="small" quaternary @click="remoteFiles = []">{{ t('files.clear') }}</NButton>
          <NButton size="small" @click="openPicker('file', 'pull', true)">{{ t('files.browseDevice') }}</NButton>
        </div>
        <div class="flex flex-1 flex-col gap-2.5 p-4">
          <div class="rounded-sm border border-dashed border-line-strong px-3 py-2.5 text-center text-[11px] text-fg3">
            {{ t('files.dropHintRemote') }}
          </div>
          <ul class="m-0 flex max-h-[260px] min-h-[120px] flex-1 list-none flex-col gap-1 overflow-y-auto rounded-sm border border-line bg-inset p-1.5">
            <li v-for="(file, index) in remoteFiles" :key="file" class="flex items-center gap-2 rounded-[6px] px-2 py-1.25 text-xs hover:bg-surface">
              <span class="flex-1 overflow-hidden text-ellipsis whitespace-nowrap" :title="file">{{ baseName(file) }}</span>
              <button
                class="h-[18px] w-[18px] shrink-0 cursor-pointer rounded-full border-0 bg-transparent text-sm leading-none text-fg3 hover:bg-err-soft hover:text-err"
                :title="t('files.remove')"
                @click="remoteFiles.splice(index, 1)"
              >
                ×
              </button>
            </li>
            <li v-if="!remoteFiles.length" class="flex justify-center rounded-[6px] py-6 text-center text-xs text-fg3">
              {{ t('files.emptyList') }}
            </li>
          </ul>
          <div class="flex items-center gap-2">
            <span class="min-w-[60px] shrink-0 whitespace-nowrap text-xs text-fg2">{{ t('files.saveTo') }}</span>
            <NInput :value="pullLocalDir" size="small" readonly :placeholder="t('files.chooseDir')" />
            <NButton size="small" @click="chooseDir">{{ t('files.chooseDir') }}</NButton>
            <NButton size="small" type="primary" :loading="pulling" :disabled="!canPull" @click="doPull">
              {{ remoteFiles.length > 1 ? t('files.pullN', { n: remoteFiles.length }) : t('files.pull') }}
            </NButton>
          </div>
        </div>
      </section>
    </div>

    <section class="rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3">
        <NIcon :size="16"><CameraOutline /></NIcon>
        <h3 class="m-0 text-[13px] font-600">{{ t('files.screenshot') }}</h3>
        <span class="ml-auto mr-2 text-[11px] text-fg3">{{ t('files.screenshotHint') }}</span>
        <NButton size="small" type="primary" :loading="shooting" :disabled="!store.current" @click="takeShot">
          {{ t('files.takeShot') }}
        </NButton>
      </div>
      <div class="flex flex-1 flex-col gap-2.5 p-4">
        <div v-if="shot" class="flex flex-col gap-2">
          <img :src="shot.dataUrl" class="max-h-[420px] max-w-full rounded-sm border border-line bg-inset object-contain" :alt="t('files.shotPreview')" />
          <div class="break-all font-mono text-[11px] text-fg3">{{ shot.path }}</div>
        </div>
        <div v-else class="rounded-sm border border-dashed border-line-strong py-6 text-center text-xs text-fg3">
          {{ t('files.noShot') }}
        </div>
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
