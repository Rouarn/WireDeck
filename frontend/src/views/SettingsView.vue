<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  AppLabelCacheInfo as fetchLabelCacheInfo,
  AppVersion,
  CheckUpdate,
  ClearAppLabelCache,
  GetHostIdentity,
  OpenExternal,
} from '../../wailsjs/go/main/App'
import { NButton, useDialog, useMessage } from 'naive-ui'
import type { model } from '../../wailsjs/go/models'
import SegmentedControl from '../components/SegmentedControl.vue'
import { useThemeStore } from '../stores/theme'
import { useDevicesStore } from '../stores/devices'
import type { ThemeMode } from '../stores/theme'

const theme = useThemeStore()
const devices = useDevicesStore()
const dialog = useDialog()
const identity = ref<model.HostIdentity | null>(null)
const cacheInfo = ref<model.LabelCacheInfo | null>(null)
const version = ref('')
const updateInfo = ref<model.UpdateInfo | null>(null)
const checking = ref(false)
const message = useMessage()

const themeOptions: { label: string; value: ThemeMode }[] = [
  { label: '跟随系统', value: 'system' },
  { label: '浅色', value: 'light' },
  { label: '深色', value: 'dark' },
]

function onChange(value: ThemeMode) {
  theme.setMode(value)
}

async function refreshCacheInfo() {
  try {
    cacheInfo.value = await fetchLabelCacheInfo()
  } catch {
    cacheInfo.value = null
  }
}

function formatSize(bytes: number) {
  if (bytes < 1024) {
    return `${bytes} B`
  }
  return `${(bytes / 1024).toFixed(1)} KB`
}

function clearCache() {
  dialog.warning({
    title: '清理应用名称缓存',
    content: '将删除所有已解析的应用名称，下次查看详情或导出时会重新解析。',
    positiveText: '清理',
    negativeText: '取消',
    onPositiveClick: async () => {
      await ClearAppLabelCache()
      await refreshCacheInfo()
    },
  })
}

onMounted(async () => {
  try {
    version.value = await AppVersion()
  } catch {
    version.value = ''
  }
  try {
    identity.value = await GetHostIdentity()
  } catch {
    identity.value = null
  }
  await refreshCacheInfo()
})

async function checkUpdate() {
  checking.value = true
  try {
    updateInfo.value = await CheckUpdate()
    if (updateInfo.value.hasUpdate) {
      message.success(`发现新版本 v${updateInfo.value.latest}`)
    } else {
      message.info('已是最新版本')
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    checking.value = false
  }
}

async function openUpdate() {
  if (updateInfo.value?.url) {
    await OpenExternal(updateInfo.value.url)
  }
}
</script>

<template>
  <div class="settings">
    <section class="panel">
      <div class="panel-head">
        <h3>版本更新</h3>
        <span class="hint">当前 v{{ version || '—' }}</span>
      </div>
      <div class="panel-body">
        <div class="row">
          <span class="label">状态</span>
          <span v-if="updateInfo">
            {{ updateInfo.hasUpdate ? `发现新版本 v${updateInfo.latest}` : '已是最新版本' }}
          </span>
          <span v-else>未检查</span>
        </div>
        <div class="row">
          <span class="label">操作</span>
          <div class="row-actions">
            <NButton size="small" :loading="checking" @click="checkUpdate">检查更新</NButton>
            <NButton
              v-if="updateInfo?.hasUpdate"
              size="small"
              type="primary"
              @click="openUpdate"
            >
              打开下载页
            </NButton>
          </div>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>外观</h3>
        <span class="hint">切换后立即生效</span>
      </div>
      <div class="panel-body">
        <SegmentedControl :options="themeOptions" :value="theme.mode" @update:value="onChange" />
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>adb 运行环境</h3>
      </div>
      <div class="panel-body">
        <div class="row">
          <span class="label">状态</span>
          <span>{{ devices.adb ? '已就绪' : '未就绪' }}</span>
        </div>
        <div class="row">
          <span class="label">路径</span>
          <code>{{ devices.adb?.binaryPath || '—' }}</code>
        </div>
        <div class="row">
          <span class="label">版本</span>
          <span>{{ devices.adb?.version || '—' }}</span>
        </div>
        <div class="row">
          <span class="label">server 端口</span>
          <span>{{ devices.adb?.serverPort || '—' }}</span>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>配对身份</h3>
        <span class="hint">手机端显示</span>
      </div>
      <div class="panel-body">
        <p class="desc">
          手机「无线调试 → 已配对设备」中显示为
          <code>{{ identity?.hostname || '—' }}@{{ identity?.lanIP || '—' }}</code>。
          @ 前是电脑主机名，@ 后是电脑局域网 IP，不是手机信息。
        </p>
        <p class="desc">该标签由 adb 在配对时上报，配对后不会自动更新。如需显示为 WireDeck，可临时修改主机名：</p>
        <pre class="cmd">sudo scutil --set HostName WireDeck   # 修改
sudo scutil --set HostName ""         # 恢复</pre>
        <p class="desc muted">修改后需在手机上移除旧配对记录并重新配对，才会更新显示。</p>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>应用名称缓存</h3>
        <span class="hint">用于列表显示中文名</span>
      </div>
      <div class="panel-body">
        <div class="row">
          <span class="label">条目数</span>
          <span>{{ cacheInfo?.count ?? 0 }} 条</span>
        </div>
        <div class="row">
          <span class="label">占用空间</span>
          <span>{{ cacheInfo ? formatSize(cacheInfo.size) : '—' }}</span>
        </div>
        <div class="row">
          <span class="label">文件位置</span>
          <code>{{ cacheInfo?.path || '—' }}</code>
        </div>
        <p class="desc muted">
          缓存只保存「包名 → 应用名」的映射，不包含 APK。手机上卸载应用后，对应记录不会自动删除，可在此清理。
        </p>
        <NButton size="small" @click="clearCache">清理缓存</NButton>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 720px;
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

.panel-head .head-action {
  display: flex;
  align-items: center;
  margin-left: auto;
}

.panel-body {
  padding: 16px;
}

.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
}

.row:last-child {
  border-bottom: none;
}

.row .label {
  width: 96px;
  color: var(--text-secondary);
}

.row code {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-all;
  color: var(--text);
}

.row-actions {
  display: flex;
  gap: 8px;
}

.desc {
  margin: 0 0 10px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.7;
}

.desc code {
  padding: 1px 6px;
  border-radius: 5px;
  background: var(--surface-inset);
  color: var(--text);
}

.desc.muted {
  color: var(--text-tertiary);
  font-size: 11px;
}

.cmd {
  margin: 0 0 10px;
  padding: 10px 12px;
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-inset);
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.6;
}
</style>
