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
  <div class="flex max-w-[720px] flex-col gap-4">
    <section class="rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('settings.update') }}</h3>
        <span class="ml-auto text-[11px] text-fg3">{{ t('settings.currentVersion', { v: version || '—' }) }}</span>
      </div>
      <div class="p-4">
        <div class="flex items-center gap-3 border-b border-line py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.status') }}</span>
          <span v-if="updateInfo">
            {{ updateInfo.hasUpdate ? t('settings.newVersion', { version: updateInfo.latest }) : t('settings.upToDate') }}
          </span>
          <span v-else>{{ t('settings.notChecked') }}</span>
        </div>
        <div class="flex items-center gap-3 py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.actions') }}</span>
          <div class="flex gap-2">
            <NButton size="small" :loading="checking" @click="checkUpdate">{{ t('settings.checkUpdate') }}</NButton>
            <NButton v-if="updateInfo?.hasUpdate" size="small" type="primary" @click="openUpdate">
              {{ t('settings.openDownload') }}
            </NButton>
          </div>
        </div>
      </div>
    </section>

    <section class="rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('settings.appearance') }}</h3>
        <span class="ml-auto text-[11px] text-fg3">{{ t('settings.appearanceHint') }}</span>
      </div>
      <div class="p-4">
        <SegmentedControl :options="themeOptions" :value="theme.mode" @update:value="onChange" />
      </div>
    </section>

    <section class="rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('settings.language') }}</h3>
        <span class="ml-auto text-[11px] text-fg3">{{ t('settings.languageHint') }}</span>
      </div>
      <div class="p-4">
        <SegmentedControl :options="languageOptions" :value="lang" @update:value="setLang" />
      </div>
    </section>

    <section class="rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('settings.adb') }}</h3>
      </div>
      <div class="p-4">
        <div class="flex items-center gap-3 border-b border-line py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.status') }}</span>
          <span>{{ devices.adb ? t('settings.ready') : t('settings.notReady') }}</span>
        </div>
        <div class="flex items-center gap-3 border-b border-line py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.adbPath') }}</span>
          <code class="min-w-0 flex-1 break-all text-fg">{{ devices.adb?.binaryPath || '—' }}</code>
        </div>
        <div class="flex items-center gap-3 border-b border-line py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.adbVersion') }}</span>
          <span>{{ devices.adb?.version || '—' }}</span>
        </div>
        <div class="flex items-center gap-3 py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.adbPort') }}</span>
          <span>{{ devices.adb?.serverPort || '—' }}</span>
        </div>
      </div>
    </section>

    <section class="rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('settings.pairing') }}</h3>
        <span class="ml-auto text-[11px] text-fg3">{{ t('settings.pairingHint') }}</span>
      </div>
      <div class="p-4">
        <p class="mb-2.5 text-xs leading-[1.7] text-fg2">
          {{ t('settings.pairingShown', { id: `${identity?.hostname || '—'}@${identity?.lanIP || '—'}` }) }}
        </p>
        <p class="mb-2.5 text-xs leading-[1.7] text-fg2">{{ t('settings.pairingHow') }}</p>
        <pre class="mb-2.5 overflow-x-auto rounded-sm border border-line bg-inset px-3 py-2.5 font-mono text-[11px] leading-relaxed text-fg2">sudo scutil --set HostName WireDeck
sudo scutil --set HostName ""</pre>
        <p class="mb-2.5 text-[11px] leading-[1.7] text-fg3">{{ t('settings.pairingUpdate') }}</p>
      </div>
    </section>

    <section class="rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-2 border-b border-line px-4 py-3.5">
        <h3 class="m-0 text-[13px] font-600">{{ t('settings.cache') }}</h3>
        <span class="ml-auto text-[11px] text-fg3">{{ t('settings.cacheHint') }}</span>
      </div>
      <div class="p-4">
        <div class="flex items-center gap-3 border-b border-line py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.cacheCount') }}</span>
          <span>{{ t('settings.cacheCountValue', { n: cacheInfo?.count ?? 0 }) }}</span>
        </div>
        <div class="flex items-center gap-3 border-b border-line py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.cacheSize') }}</span>
          <span>{{ cacheInfo ? formatSize(cacheInfo.size) : '—' }}</span>
        </div>
        <div class="flex items-center gap-3 border-b border-line py-2 text-xs">
          <span class="w-24 text-fg2">{{ t('settings.cachePath') }}</span>
          <code class="min-w-0 flex-1 break-all text-fg">{{ cacheInfo?.path || '—' }}</code>
        </div>
        <p class="mb-2.5 mt-2 text-[11px] leading-[1.7] text-fg3">
          {{ t('settings.cacheNote') }}
        </p>
        <NButton size="small" @click="clearCache">{{ t('settings.cacheClear') }}</NButton>
      </div>
    </section>
  </div>
</template>
