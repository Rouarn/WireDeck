<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NDrawer,
  NDrawerContent,
  NDropdown,
  NIcon,
  NInput,
  NPopconfirm,
  NSpin,
  useDialog,
  useMessage,
} from 'naive-ui'
import { SearchOutline } from '@vicons/ionicons5'
import {
  AppDetail as fetchAppDetail,
  ClearAppData,
  ExportApp,
  ForceStopApp,
  LaunchApp,
  ListApps,
  PickDirectory,
  UninstallApp,
} from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import VirtualList from '../components/VirtualList.vue'
import SegmentedControl from '../components/SegmentedControl.vue'
import { useDevicesStore } from '../stores/devices'

interface AppProgressPayload {
  pkg: string
  stage: string
  percent: number
  message: string
}

const store = useDevicesStore()
const message = useMessage()
const dialog = useDialog()

const apps = ref<model.AppSummary[]>([])
const loading = ref(false)
const search = ref('')
const scope = ref<'user' | 'system' | 'all'>('user')

const scopeOptions: { label: string; value: 'user' | 'system' | 'all' }[] = [
  { label: '第三方', value: 'user' },
  { label: '系统', value: 'system' },
  { label: '全部', value: 'all' },
]

function setScope(value: 'user' | 'system' | 'all') {
  if (scope.value === value) {
    return
  }
  scope.value = value
  void load()
}
const showDetail = ref(false)
const detailLoading = ref(false)
const detail = ref<model.AppDetail | null>(null)
const pendingDetailPkg = ref('')
const detailProgress = ref(0)
const exportProgress = ref(0)
const exporting = ref(false)

const visibleApps = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) {
    return apps.value
  }
  return apps.value.filter(
    (app) =>
      app.package.toLowerCase().includes(keyword) ||
      (app.label || '').toLowerCase().includes(keyword),
  )
})

const moreOptions = [
  { label: '启动应用', key: 'launch' },
  { label: '强制停止', key: 'stop' },
  { label: '清除数据', key: 'clear' },
]

const avatarPalette = [
  { bg: 'var(--primary-soft)', fg: 'var(--primary-ink)' },
  { bg: 'var(--success-soft)', fg: 'var(--success)' },
  { bg: 'var(--warning-soft)', fg: 'var(--warning)' },
  { bg: 'var(--danger-soft)', fg: 'var(--danger)' },
]

function avatarStyle(pkg: string) {
  let hash = 0
  for (let i = 0; i < pkg.length; i += 1) {
    hash = (hash * 31 + pkg.charCodeAt(i)) >>> 0
  }
  return avatarPalette[hash % avatarPalette.length]
}

function avatarText(item: model.AppSummary) {
  const base = item.label || item.package
  return (base[0] || '?').toUpperCase()
}

const showAllPerms = ref(false)

const visiblePerms = computed(() => {
  const perms = detail.value?.permissions ?? []
  return showAllPerms.value ? perms : perms.slice(0, 6)
})

function avatarLetter(value: model.AppDetail) {
  const text = value.label || value.package
  return (text[0] || '?').toUpperCase()
}

async function load() {
  if (!store.current) {
    apps.value = []
    return
  }
  loading.value = true
  try {
    apps.value = await ListApps(store.current.serial, scope.value)
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    loading.value = false
  }
}

async function openDetail(pkg: string) {
  if (!store.current) {
    return
  }
  showDetail.value = true
  detailLoading.value = true
  detail.value = null
  pendingDetailPkg.value = pkg
  detailProgress.value = 0
  try {
    detail.value = await fetchAppDetail(store.current.serial, pkg)
    // 详情解析会得到应用显示名，同步回列表，避免用户再手动刷新。
    if (detail.value.label) {
      const target = apps.value.find((item) => item.package === pkg)
      if (target) {
        target.label = detail.value.label
      }
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    detailLoading.value = false
  }
}

async function exportApp(pkg: string) {
  if (!store.current) {
    return
  }
  try {
    const dir = await PickDirectory()
    if (!dir) {
      return
    }
    exporting.value = true
    exportProgress.value = 0
    await ExportApp(store.current.serial, pkg, dir)
    message.success(`已导出到 ${dir}`)
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    exporting.value = false
  }
}

function onAppProgress(p: AppProgressPayload) {
  if (p.stage === 'label' && p.pkg === pendingDetailPkg.value && detailLoading.value) {
    detailProgress.value = Math.max(detailProgress.value, p.percent)
  }
  if (p.stage === 'export' && exporting.value) {
    exportProgress.value = Math.max(exportProgress.value, p.percent)
  }
}

async function uninstall(pkg: string) {
  if (!store.current) {
    return
  }
  try {
    const result = await UninstallApp(store.current.serial, pkg, false)
    if (result.ok) {
      message.success(`${pkg} 已卸载`)
      await load()
    } else {
      message.error(result.message)
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function uninstallAndClose(pkg: string) {
  showDetail.value = false
  await uninstall(pkg)
}

async function onMore(key: string, pkg: string) {
  if (!store.current) {
    return
  }

  // 清除数据不可撤销，单独二次确认。
  if (key === 'clear') {
    const serial = store.current.serial
    dialog.warning({
      title: '清除应用数据',
      content: `确认清除 ${pkg} 的全部数据？此操作不可撤销。`,
      positiveText: '清除数据',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          const result = await ClearAppData(serial, pkg)
          if (result.ok) {
            message.success(result.message)
          } else {
            message.error(result.message)
          }
        } catch (e) {
          message.error(e instanceof Error ? e.message : String(e))
        }
      },
    })
    return
  }

  try {
    let result: model.AppOpResult
    if (key === 'launch') {
      result = await LaunchApp(store.current.serial, pkg)
    } else {
      result = await ForceStopApp(store.current.serial, pkg)
    }
    if (result.ok) {
      message.success(result.message)
    } else {
      message.error(result.message)
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

onMounted(() => {
  EventsOn('app:progress', (p: AppProgressPayload) => onAppProgress(p))
  void load()
})
</script>

<template>
  <div class="apps">
    <NAlert v-if="!store.current" type="warning" class="alert">
      请先在「设备」页选择一台设备。
    </NAlert>

    <div class="toolbar">
      <NInput
        v-model:value="search"
        size="small"
        placeholder="搜索应用名称或包名"
        clearable
        class="search-input"
        style="width: 260px"
      >
        <template #prefix>
          <NIcon :size="14"><SearchOutline /></NIcon>
        </template>
      </NInput>
      <SegmentedControl :options="scopeOptions" :value="scope" @update:value="setScope" />
      <span class="count">{{ visibleApps.length }} / {{ apps.length }}</span>
      <div class="spacer"></div>
      <NButton size="small" :loading="loading" @click="load">刷新</NButton>
    </div>

    <div v-if="exporting" class="export-bar">
      <span class="export-label">正在导出…</span>
      <div class="export-progress"><i :style="{ width: `${exportProgress}%` }"></i></div>
      <span class="export-percent">{{ exportProgress }}%</span>
    </div>

    <section class="panel">
      <div class="table-head">
        <span>应用</span>
        <span class="head-ops">操作</span>
      </div>
      <div class="table-body">
        <VirtualList v-if="visibleApps.length" :items="visibleApps" :item-height="48">
          <template #default="{ item }">
            <div class="app-row">
              <div class="app-cell">
                <div
                  class="avatar"
                  :style="{
                    background: avatarStyle(item.package).bg,
                    color: avatarStyle(item.package).fg,
                  }"
                >
                  {{ avatarText(item) }}
                </div>
                <div class="app-info">
                  <div class="app-name">{{ item.label || item.package }}</div>
                  <div v-if="item.label" class="app-pkg">{{ item.package }}</div>
                </div>
              </div>
              <div class="ops">
                <NButton size="tiny" quaternary @click="openDetail(item.package)">详情</NButton>
                <NButton size="tiny" quaternary @click="exportApp(item.package)">导出</NButton>
                <NPopconfirm @positive-click="uninstall(item.package)">
                  <template #trigger>
                    <NButton size="tiny" quaternary type="error">卸载</NButton>
                  </template>
                  确认卸载 {{ item.package }}？此操作不可撤销。
                </NPopconfirm>
                <NDropdown :options="moreOptions" @select="(key) => onMore(String(key), item.package)">
                  <NButton size="tiny" quaternary>更多</NButton>
                </NDropdown>
              </div>
            </div>
          </template>
        </VirtualList>
        <div v-else class="empty">
          {{ store.current ? '没有匹配的应用' : '请先选择设备' }}
        </div>
      </div>
    </section>

    <NDrawer v-model:show="showDetail" :width="600">
      <NDrawerContent title="应用详情" closable :native-scrollbar="false">
        <NSpin :show="detailLoading">
          <div v-if="detail" class="detail">
            <div class="detail-hero">
              <div class="hero-avatar">{{ avatarLetter(detail) }}</div>
              <div class="hero-main">
                <div class="hero-name">{{ detail.label || detail.package }}</div>
                <div class="hero-pkg">{{ detail.package }}</div>
              </div>
              <div class="hero-actions">
                <NButton size="small" @click="onMore('launch', detail.package)">启动</NButton>
                <NButton size="small" @click="exportApp(detail.package)">导出</NButton>
                <NPopconfirm @positive-click="uninstallAndClose(detail.package)">
                  <template #trigger>
                    <NButton size="small" type="error" ghost>卸载</NButton>
                  </template>
                  确认卸载 {{ detail.package }}？此操作不可撤销。
                </NPopconfirm>
              </div>
            </div>

            <div class="detail-grid">
              <div class="grid-item">
                <span class="grid-label">版本</span>
                <span class="grid-value">{{ detail.versionName }} ({{ detail.versionCode }})</span>
              </div>
              <div class="grid-item">
                <span class="grid-label">SDK</span>
                <span class="grid-value">min {{ detail.minSdk }} · target {{ detail.targetSdk }}</span>
              </div>
              <div class="grid-item">
                <span class="grid-label">首次安装</span>
                <span class="grid-value">{{ detail.firstInstall || '—' }}</span>
              </div>
              <div class="grid-item">
                <span class="grid-label">最近更新</span>
                <span class="grid-value">{{ detail.lastUpdate || '—' }}</span>
              </div>
              <div class="grid-item wide">
                <span class="grid-label">数据目录</span>
                <span class="grid-value mono">{{ detail.dataDir || '—' }}</span>
              </div>
            </div>

            <div class="detail-section">
              <div class="section-header">
                <span>已授予权限 <em>{{ detail.permissions.length }}</em></span>
                <NButton
                  v-if="detail.permissions.length > 6"
                  text
                  size="tiny"
                  @click="showAllPerms = !showAllPerms"
                >
                  {{ showAllPerms ? '收起' : '展开全部' }}
                </NButton>
              </div>
              <ul v-if="detail.permissions.length" class="perm-list">
                <li v-for="perm in visiblePerms" :key="perm">{{ perm }}</li>
              </ul>
              <div v-else class="empty-perm">该应用没有已授予的运行时权限</div>
            </div>
          </div>
          <div v-else-if="detailLoading" class="detail-loading">
            <div class="loading-title">正在读取应用信息…</div>
            <div class="loading-progress">
              <i :style="{ width: `${detailProgress}%` }"></i>
            </div>
            <div class="loading-hint">{{ detailProgress }}%</div>
          </div>
        </NSpin>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.apps {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.alert {
  margin-bottom: 16px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.toolbar .count {
  color: var(--text-tertiary);
  font-size: 11px;
}

/* 让工具栏控件与 32px 的分段控件保持同一高度 */
.toolbar :deep(.n-input) {
  height: 32px;
}

.toolbar :deep(.n-input .n-input__input-el),
.toolbar :deep(.n-input .n-input__placeholder) {
  height: 32px;
  line-height: 32px;
}

.toolbar :deep(.n-button) {
  height: 32px;
}

.spacer {
  flex: 1;
}

.panel {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.table-head {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: 1fr var(--ops-col, 260px);
  gap: 12px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--border);
  color: var(--text-tertiary);
  font-size: 10px;
  font-weight: 500;
  text-transform: uppercase;
}

.table-head .head-ops {
  text-align: right;
}

.table-body {
  flex: 1;
  min-height: 0;
}

.app-row {
  display: grid;
  grid-template-columns: 1fr var(--ops-col, 260px);
  gap: 12px;
  align-items: center;
  height: 48px;
  padding: 0 14px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
}

.app-row:hover {
  background: var(--surface-inset);
}

.app-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
}

.app-info {
  min-width: 0;
}

.app-name {
  overflow: hidden;
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-pkg {
  overflow: hidden;
  color: var(--text-tertiary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}

.empty {
  padding: 40px 0;
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}

.export-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding: 8px 12px;
  border: 1px solid var(--primary-border);
  border-radius: var(--radius-sm);
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 12px;
}

.export-progress {
  flex: 1;
  height: 5px;
  overflow: hidden;
  border-radius: 3px;
  background: var(--border);
}

.export-progress > i {
  display: block;
  height: 100%;
  background: var(--primary);
  transition: width 0.2s ease;
}

.export-percent {
  width: 36px;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.detail-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 240px;
  color: var(--text-secondary);
  font-size: 12px;
}

.loading-title {
  font-size: 13px;
}

.loading-progress {
  width: 220px;
  height: 6px;
  overflow: hidden;
  border-radius: 3px;
  background: var(--border);
}

.loading-progress > i {
  display: block;
  height: 100%;
  background: var(--primary);
  transition: width 0.2s ease;
}

.loading-hint {
  color: var(--text-tertiary);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.detail-hero {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: linear-gradient(135deg, var(--surface-inset), var(--surface));
}

.hero-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 20px;
  font-weight: 700;
}

.hero-main {
  flex: 1;
  min-width: 0;
}

.hero-name {
  font-size: 15px;
  font-weight: 600;
}

.hero-pkg {
  margin-top: 2px;
  color: var(--text-tertiary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  word-break: break-all;
}

.hero-actions {
  display: flex;
  gap: 8px;
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-top: 14px;
}

.grid-item {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
}

.grid-item.wide {
  grid-column: span 2;
}

.grid-label {
  display: block;
  color: var(--text-tertiary);
  font-size: 10px;
}

.grid-value {
  display: block;
  margin-top: 3px;
  font-size: 12px;
  word-break: break-all;
}

.grid-value.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
}

.detail-section {
  margin-top: 18px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
}

.section-header em {
  margin-left: 4px;
  color: var(--text-tertiary);
  font-style: normal;
  font-weight: 400;
}

.perm-list {
  margin: 0;
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  list-style: none;
}

.perm-list li {
  padding: 6px 10px;
  border-bottom: 1px solid var(--border);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  word-break: break-all;
}

.perm-list li:nth-child(even) {
  background: var(--surface-inset);
}

.perm-list li:last-child {
  border-bottom: none;
}

.empty-perm {
  padding: 16px;
  border: 1px dashed var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}
</style>
