<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NIcon, NInput, NSpin, useMessage } from 'naive-ui'
import { CameraOutline, CloudDownloadOutline, CloudUploadOutline } from '@vicons/ionicons5'
import { PickDirectory, PickFile, PullFile, PushFile, Screenshot } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useDevicesStore } from '../stores/devices'

interface FileProgressPayload {
  stage: string
  percent: number
  message: string
}

const store = useDevicesStore()
const message = useMessage()

const shot = ref<model.ScreenshotResult | null>(null)
const shooting = ref(false)

const pushLocal = ref('')
const pushRemote = ref('/sdcard/Download/')
const pushing = ref(false)

const pullRemote = ref('/sdcard/Download/')
const pullLocalDir = ref('')
const pulling = ref(false)

const transfer = ref({ active: false, label: '', percent: 0 })

const busy = computed(() => shooting.value || pushing.value || pulling.value)

function onProgress(p: FileProgressPayload) {
  transfer.value.active = true
  transfer.value.label = p.message
  transfer.value.percent = Math.max(transfer.value.percent, p.percent)
}

async function takeShot() {
  if (!store.current) {
    message.warning('请先在「设备」页选择设备')
    return
  }
  shooting.value = true
  try {
    shot.value = await Screenshot(store.current.serial)
    message.success('截图已保存')
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    shooting.value = false
  }
}

async function chooseLocal() {
  try {
    const path = await PickFile()
    if (path) {
      pushLocal.value = path
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

async function doPush() {
  if (!store.current) {
    message.warning('请先在「设备」页选择设备')
    return
  }
  if (!pushLocal.value || !pushRemote.value.trim()) {
    message.warning('请选择本地文件并填写设备上的目标路径')
    return
  }

  pushing.value = true
  transfer.value = { active: true, label: '推送 0%', percent: 0 }
  try {
    await PushFile(store.current.serial, pushLocal.value, pushRemote.value.trim())
    message.success('推送完成')
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    pushing.value = false
    transfer.value.active = false
  }
}

async function doPull() {
  if (!store.current) {
    message.warning('请先在「设备」页选择设备')
    return
  }
  if (!pullRemote.value.trim() || !pullLocalDir.value) {
    message.warning('请填写设备上的文件路径并选择保存目录')
    return
  }

  pulling.value = true
  transfer.value = { active: true, label: '拉取 0%', percent: 0 }
  try {
    await PullFile(store.current.serial, pullRemote.value.trim(), pullLocalDir.value)
    message.success('拉取完成')
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    pulling.value = false
    transfer.value.active = false
  }
}

onMounted(() => {
  EventsOn('file:progress', (p: FileProgressPayload) => onProgress(p))
})
</script>

<template>
  <div class="files">
    <NAlert v-if="!store.current" type="warning" class="alert">
      请先在「设备」页选择一台设备。
    </NAlert>

    <div v-if="transfer.active" class="transfer-bar">
      <span class="transfer-label">{{ transfer.label }}</span>
      <div class="transfer-progress"><i :style="{ width: `${transfer.percent}%` }"></i></div>
      <span class="transfer-percent">{{ transfer.percent }}%</span>
    </div>

    <section class="panel">
      <div class="panel-head">
        <NIcon :size="16"><CameraOutline /></NIcon>
        <h3>截图</h3>
        <span class="hint">保存到 ~/Pictures/WireDeck</span>
        <NButton size="small" type="primary" :loading="shooting" :disabled="!store.current" @click="takeShot">
          截取屏幕
        </NButton>
      </div>
      <div class="panel-body">
        <div v-if="shot" class="shot">
          <img :src="shot.dataUrl" class="shot-image" alt="截图预览" />
          <div class="shot-path">{{ shot.path }}</div>
        </div>
        <div v-else class="shot-empty">还没有截图</div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <NIcon :size="16"><CloudUploadOutline /></NIcon>
        <h3>推送文件到设备</h3>
        <span class="hint">本地 → 手机</span>
      </div>
      <div class="panel-body">
        <div class="form-row">
          <span class="form-label">本地文件</span>
          <NInput :value="pushLocal" size="small" readonly placeholder="点击右侧选择文件" class="grow" />
          <NButton size="small" @click="chooseLocal">选择</NButton>
        </div>
        <div class="form-row">
          <span class="form-label">目标路径</span>
          <NInput v-model:value="pushRemote" size="small" placeholder="/sdcard/Download/" class="grow" />
          <NButton size="small" type="primary" :loading="pushing" :disabled="!store.current" @click="doPush">
            推送
          </NButton>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <NIcon :size="16"><CloudDownloadOutline /></NIcon>
        <h3>从设备拉取文件</h3>
        <span class="hint">手机 → 本地</span>
      </div>
      <div class="panel-body">
        <div class="form-row">
          <span class="form-label">设备路径</span>
          <NInput v-model:value="pullRemote" size="small" placeholder="/sdcard/Download/example.txt" class="grow" />
        </div>
        <div class="form-row">
          <span class="form-label">保存到</span>
          <NInput :value="pullLocalDir" size="small" readonly placeholder="点击右侧选择目录" class="grow" />
          <NButton size="small" @click="chooseDir">选择</NButton>
          <NButton size="small" type="primary" :loading="pulling" :disabled="!store.current" @click="doPull">
            拉取
          </NButton>
        </div>
      </div>
    </section>
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
  width: 90px;
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

.panel {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
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
  padding: 16px;
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

.form-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
}

.form-row:first-child {
  margin-top: 0;
}

.form-label {
  flex: 0 0 72px;
  color: var(--text-secondary);
  font-size: 12px;
}

.grow {
  flex: 1;
  min-width: 0;
}
</style>
