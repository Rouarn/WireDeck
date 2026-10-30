<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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
import { t, useLang, type Lang } from '../i18n'

const theme = useThemeStore()
const devices = useDevicesStore()
const dialog = useDialog()
const identity = ref<model.HostIdentity | null>(null)
const cacheInfo = ref<model.LabelCacheInfo | null>(null)
const version = ref('')
const updateInfo = ref<model.UpdateInfo | null>(null)
const checking = ref(false)
const message = useMessage()

const { lang, setLang } = useLang()

const themeOptions = computed<{ label: string; value: ThemeMode }[]>(() => [
  { label: t('settings.theme.system'), value: 'system' },
  { label: t('settings.theme.light'), value: 'light' },
  { label: t('settings.theme.dark'), value: 'dark' },
])

const languageOptions: { label: string; value: Lang }[] = [
  { label: '中文', value: 'zh' },
  { label: 'English', value: 'en' },
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
    title: t('settings.cacheDialogTitle'),
    content: t('settings.cacheDialogContent'),
    positiveText: t('settings.cacheDialogOk'),
    negativeText: t('settings.cacheDialogCancel'),
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
      message.success(t('settings.newVersion', { version: updateInfo.value.latest }))
    } else {
      message.info(t('settings.upToDate'))
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
        <h3>{{ t('settings.update') }}</h3>
        <span class="hint">{{ t('settings.currentVersion', { v: version || '—' }) }}</span>
      </div>
      <div class="panel-body">
        <div class="row">
          <span class="label">状态</span>
          <span v-if="updateInfo">
            {{ updateInfo.hasUpdate ? t('settings.newVersion', { version: updateInfo.latest }) : t('settings.upToDate') }}
          </span>
          <span v-else>{{ t('settings.notChecked') }}</span>
        </div>
        <div class="row">
          <span class="label">操作</span>
          <div class="row-actions">
            <NButton size="small" :loading="checking" @click="checkUpdate">{{ t('settings.checkUpdate') }}</NButton>
            <NButton
              v-if="updateInfo?.hasUpdate"
              size="small"
              type="primary"
              @click="openUpdate"
            >
              {{ t('settings.openDownload') }}
            </NButton>
          </div>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>{{ t('settings.appearance') }}</h3>
        <span class="hint">{{ t('settings.appearanceHint') }}</span>
      </div>
      <div class="panel-body">
        <SegmentedControl :options="themeOptions" :value="theme.mode" @update:value="onChange" />
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>{{ t('settings.language') }}</h3>
        <span class="hint">{{ t('settings.languageHint') }}</span>
      </div>
      <div class="panel-body">
        <SegmentedControl :options="languageOptions" :value="lang" @update:value="setLang" />
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>{{ t('settings.adb') }}</h3>
      </div>
      <div class="panel-body">
        <div class="row">
          <span class="label">状态</span>
          <span>{{ devices.adb ? t('settings.ready') : t('settings.notReady') }}</span>
        </div>
        <div class="row">
          <span class="label">{{ t('settings.adbPath') }}</span>
          <code>{{ devices.adb?.binaryPath || '—' }}</code>
        </div>
        <div class="row">
          <span class="label">{{ t('settings.adbVersion') }}</span>
          <span>{{ devices.adb?.version || '—' }}</span>
        </div>
        <div class="row">
          <span class="label">{{ t('settings.adbPort') }}</span>
          <span>{{ devices.adb?.serverPort || '—' }}</span>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>{{ t('settings.pairing') }}</h3>
        <span class="hint">{{ t('settings.pairingHint') }}</span>
      </div>
      <div class="panel-body">
        <p class="desc">
          {{ t('settings.pairingShown', { id: `${identity?.hostname || '—'}@${identity?.lanIP || '—'}` }) }}
        </p>
        <p class="desc">{{ t('settings.pairingHow') }}</p>
        <pre class="cmd">sudo scutil --set HostName WireDeck
sudo scutil --set HostName ""</pre>
        <p class="desc muted">{{ t('settings.pairingUpdate') }}</p>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>{{ t('settings.cache') }}</h3>
        <span class="hint">{{ t('settings.cacheHint') }}</span>
      </div>
      <div class="panel-body">
        <div class="row">
          <span class="label">{{ t('settings.cacheCount') }}</span>
          <span>{{ t('settings.cacheCountValue', { n: cacheInfo?.count ?? 0 }) }}</span>
        </div>
        <div class="row">
          <span class="label">{{ t('settings.cacheSize') }}</span>
          <span>{{ cacheInfo ? formatSize(cacheInfo.size) : '—' }}</span>
        </div>
        <div class="row">
          <span class="label">{{ t('settings.cachePath') }}</span>
          <code>{{ cacheInfo?.path || '—' }}</code>
        </div>
        <p class="desc muted">
          {{ t('settings.cacheNote') }}
        </p>
        <NButton size="small" @click="clearCache">{{ t('settings.cacheClear') }}</NButton>
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
