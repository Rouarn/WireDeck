<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { NIcon } from 'naive-ui'
import {
  CloudDownloadOutline,
  CodeSlashOutline,
  DocumentTextOutline,
  FolderOutline,
  GridOutline,
  PhonePortraitOutline,
  SettingsOutline,
} from '@vicons/ionicons5'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { model } from '../../wailsjs/go/models'
import { useDevicesStore } from '../stores/devices'
import { t } from '../i18n'

const route = useRoute()
const store = useDevicesStore()

const nav = [
  { to: '/devices', key: 'nav.devices', icon: PhonePortraitOutline, badge: true },
  { to: '/install', key: 'nav.install', icon: CloudDownloadOutline, badge: false },
  { to: '/apps', key: 'nav.apps', icon: GridOutline, badge: false },
  { to: '/files', key: 'nav.files', icon: FolderOutline, badge: false },
  { to: '/logs', key: 'nav.logs', icon: DocumentTextOutline, badge: false },
  { to: '/shell', key: 'nav.shell', icon: CodeSlashOutline, badge: false },
  { to: '/settings', key: 'nav.settings', icon: SettingsOutline, badge: false },
]

const adbLabel = computed(() =>
  store.adb ? `adb ${store.adb.version} · :${store.adb.serverPort}` : t('common.adbNotReady'),
)
const deviceCount = computed(() => store.list.length)

// 这些页面内部自带滚动（虚拟列表），外层不再滚动，避免出现双滚动条。
const fullHeightPages = ['/apps', '/logs', '/shell']
const isFullHeight = computed(() => fullHeightPages.includes(route.path))

onMounted(async () => {
  await store.refresh()
  EventsOn('device:changed', (devices: model.Device[]) => store.apply(devices))
})
</script>

<template>
  <div class="flex h-screen overflow-hidden">
    <aside class="flex w-[232px] shrink-0 flex-col border-r border-line bg-muted p-3">
      <div class="flex items-center gap-2.5 px-2 pt-1 pb-4.5">
        <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-brand text-sm font-700 text-white">
          W
        </div>
        <div>
          <div class="text-sm font-600">WireDeck</div>
          <div class="text-11px text-fg3">{{ t('app.tagline') }}</div>
        </div>
      </div>

      <nav class="flex flex-col gap-0.5">
        <RouterLink
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2.5 rounded-sm px-2.5 py-2 text-13px no-underline"
          :class="route.path === item.to ? 'bg-brand-soft font-600 text-brand-ink' : 'text-fg2 hover:text-fg'"
        >
          <NIcon :size="16" :component="item.icon" />
          <span>{{ t(item.key) }}</span>
          <span
            v-if="item.badge && deviceCount"
            class="ml-auto h-[18px] min-w-[18px] rounded-[9px] bg-brand px-1.25 text-center text-11px leading-[18px] text-white"
          >
            {{ deviceCount }}
          </span>
        </RouterLink>
      </nav>

      <div class="mt-auto pt-3">
        <div class="flex items-center gap-2 rounded-sm border border-line bg-surface px-2.5 py-2 text-11px text-fg2">
          <span
            class="h-[7px] w-[7px] rounded-full"
            :class="store.adb ? 'bg-[#12b76a] shadow-[0_0_0_3px_rgba(18,183,106,0.12)]' : 'bg-fg3'"
          ></span>
          <span>{{ adbLabel }}</span>
        </div>
      </div>
    </aside>

    <main class="flex min-w-0 flex-1 flex-col bg-bg">
      <header class="flex h-14 shrink-0 items-center gap-3 border-b border-line bg-surface px-5">
        <div class="text-[15px] font-600">{{ t(String(route.meta.titleKey || 'nav.devices')) }}</div>
        <div class="flex-1"></div>
        <div class="flex items-center gap-2 rounded-sm border border-brand-line bg-brand-soft px-2.5 py-1.5 text-xs text-fg2">
          <span
            class="h-[7px] w-[7px] rounded-full"
            :class="store.current ? 'bg-[#12b76a]' : 'bg-fg3'"
          ></span>
          <span v-if="store.current">
            <strong class="font-600 text-brand-ink">{{ store.current.model || store.current.serial }}</strong>
            · {{ store.current.transport === 'wifi' ? 'Wi-Fi' : 'USB' }}
          </span>
          <span v-else>{{ t('common.unknown') }}</span>
        </div>
        <button
          class="h-8 rounded-sm border border-line-strong bg-surface px-3 text-xs text-fg hover:bg-muted disabled:text-fg3"
          :disabled="store.loading"
          @click="store.refresh()"
        >
          {{ store.loading ? t('common.refreshing') : t('common.refresh') }}
        </button>
      </header>

      <div
        class="flex-1 px-6 pt-5 pb-8"
        :class="isFullHeight ? 'flex flex-col overflow-hidden pb-5' : 'overflow-auto'"
      >
        <RouterView />
      </div>
    </main>
  </div>
</template>
