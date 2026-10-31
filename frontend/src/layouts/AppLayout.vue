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
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2C12.9 7.4 16.6 11.1 22 12C16.6 12.9 12.9 16.6 12 22C11.1 16.6 7.4 12.9 2 12C7.4 11.1 11.1 7.4 12 2Z"/></svg></div>
        <div>
          <div class="brand-name">WireDeck</div>
          <div class="brand-version">{{ t('app.tagline') }}</div>
        </div>
      </div>

      <nav class="nav">
        <RouterLink
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          active-class="active"
        >
          <NIcon :size="16" :component="item.icon" />
          <span>{{ t(item.key) }}</span>
          <span v-if="item.badge && deviceCount" class="nav-badge">{{ deviceCount }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <div class="adb-status">
          <span class="dot" :class="{ off: !store.adb }"></span>
          <span>{{ adbLabel }}</span>
        </div>
      </div>
    </aside>

    <main class="main">
      <header class="topbar">
        <div class="page-title">{{ t(String(route.meta.titleKey || 'nav.devices')) }}</div>
        <div class="topbar-spacer"></div>
        <div class="device-switcher">
          <span class="dot" :class="{ off: !store.current }"></span>
          <span v-if="store.current">
            <strong>{{ store.current.model || store.current.serial }}</strong>
            · {{ store.current.transport === 'wifi' ? 'Wi-Fi' : 'USB' }}
          </span>
          <span v-else>{{ t('common.unknown') }}</span>
        </div>
        <button class="topbar-btn" :disabled="store.loading" @click="store.refresh()">
          {{ store.loading ? t('common.refreshing') : t('common.refresh') }}
        </button>
      </header>

      <div class="content" :class="{ 'content-full': isFullHeight }">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  display: flex;
  flex-direction: column;
  flex: 0 0 232px;
  width: 232px;
  padding: 16px 12px;
  border-right: 1px solid var(--border);
  background: var(--surface-muted);
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 8px 18px;
}

.brand-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: #fff;
  border-radius: 8px;
  background: var(--primary);
  color: #fff;
  font-size: 14px;
  font-weight: 700;
}

.brand-mark svg {
  width: 16px;
  height: 16px;
  fill: currentColor;
}

.brand-name {
  font-size: 14px;
  font-weight: 600;
}

.brand-version {
  color: var(--text-tertiary);
  font-size: 11px;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 13px;
  text-decoration: none;
}

.nav-item:hover {
  color: var(--text);
}

.nav-item.active {
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-weight: 600;
}

.nav-badge {
  min-width: 18px;
  height: 18px;
  margin-left: auto;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--primary);
  color: #fff;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
}

.sidebar-footer {
  margin-top: auto;
  padding-top: 12px;
}

.adb-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text-secondary);
  font-size: 11px;
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #12b76a;
  box-shadow: 0 0 0 3px rgba(18, 183, 106, 0.12);
}

.dot.off {
  background: var(--text-tertiary);
  box-shadow: none;
}

.main {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  background: var(--bg);
}

.topbar {
  display: flex;
  align-items: center;
  flex: 0 0 56px;
  gap: 12px;
  height: 56px;
  padding: 0 20px;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
}

.page-title {
  font-size: 15px;
  font-weight: 600;
}

.topbar-spacer {
  flex: 1;
}

.device-switcher {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid var(--primary-border);
  border-radius: var(--radius-sm);
  background: var(--primary-soft);
  color: var(--text-secondary);
  font-size: 12px;
}

.device-switcher strong {
  color: var(--primary-ink);
  font-weight: 600;
}

.topbar-btn {
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text);
  font-size: 12px;
  cursor: pointer;
}

.topbar-btn:disabled {
  color: var(--text-tertiary);
  cursor: default;
}

.content {
  flex: 1;
  overflow: auto;
  padding: 20px 24px 32px;
}

.content-full {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding-bottom: 20px;
}
</style>
