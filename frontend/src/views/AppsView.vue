<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
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
import { useLabelStore } from '../stores/labels'
import { t } from '../i18n'

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
const labels = useLabelStore()
const search = ref('')
const scope = ref<'user' | 'system' | 'all'>('user')

const scopeOptions = computed<{ label: string; value: 'user' | 'system' | 'all' }[]>(() => [
  { label: t('apps.scopeUser'), value: 'user' },
  { label: t('apps.scopeSystem'), value: 'system' },
  { label: t('apps.scopeAll'), value: 'all' },
])

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
  { label: t('apps.menuLaunch'), key: 'launch' },
  { label: t('apps.menuStop'), key: 'stop' },
  { label: t('apps.menuClear'), key: 'clear' },
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
    applyLabels()
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    loading.value = false
  }
}

function applyLabels() {
  const mapping = labels.job.results
  for (const app of apps.value) {
    const label = mapping[app.package]
    if (label) {
      app.label = label
    }
  }
}

watch(
  () => labels.job.results,
  () => applyLabels(),
  { deep: true },
)

async function loadLabels() {
  if (!store.current) {
    message.warning(t('apps.selectDeviceShort'))
    return
  }

  const pkgs = apps.value.filter((app) => !app.label).map((app) => app.package)
  if (!pkgs.length) {
    message.info(t('apps.allHaveNames'))
    return
  }

  await labels.load(store.current.serial, pkgs)
  applyLabels()
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
    message.success(t('apps.exportedTo', { dir }))
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
      message.success(t('apps.uninstalled', { pkg }))
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
      title: t('apps.clearDataTitle'),
      content: t('apps.clearDataContent', { pkg }),
      positiveText: t('apps.clearDataOk'),
      negativeText: t('common.cancel'),
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
  <div class="flex min-h-0 flex-1 flex-col">
    <NAlert v-if="!store.current" type="warning" class="mb-4">
      {{ t('apps.selectDevice') }}
    </NAlert>

    <div class="mb-3 flex items-center gap-3">
      <NInput
        v-model:value="search"
        size="small"
        :placeholder="t('apps.searchPlaceholder')"
        clearable
        style="width: 260px"
      >
        <template #prefix>
          <NIcon :size="14"><SearchOutline /></NIcon>
        </template>
      </NInput>
      <SegmentedControl :options="scopeOptions" :value="scope" @update:value="setScope" />
      <span class="text-[11px] text-fg3">{{ visibleApps.length }} / {{ apps.length }}</span>
      <div class="flex-1"></div>
      <NButton size="small" :loading="labels.job.running" @click="loadLabels">
        {{ labels.job.running ? labels.job.message || t('apps.loadingNames') : t('apps.loadNames') }}
      </NButton>
      <NButton size="small" :loading="loading" @click="load">{{ t('apps.refresh') }}</NButton>
    </div>

    <div
      v-if="exporting"
      class="mb-3 flex items-center gap-2.5 rounded-sm border border-brand-line bg-brand-soft px-3 py-2 text-xs text-brand-ink"
    >
      <span class="w-[110px] whitespace-nowrap">{{ t('apps.exporting') }}</span>
      <div class="h-[5px] flex-1 overflow-hidden rounded-[3px] bg-line">
        <i class="block h-full bg-brand transition-[width] duration-200" :style="{ width: `${exportProgress}%` }"></i>
      </div>
      <span class="w-9 text-right tabular-nums">{{ exportProgress }}%</span>
    </div>

    <section class="flex min-h-0 flex-1 flex-col overflow-hidden rounded border border-line bg-surface shadow-sm">
      <div class="grid grid-cols-[1fr_260px] gap-3 border-b border-line px-3.5 py-2 text-[10px] font-500 uppercase text-fg3">
        <span>{{ t('apps.colApp') }}</span>
        <span class="text-right">{{ t('apps.colOps') }}</span>
      </div>
      <div class="min-h-0 flex-1">
        <VirtualList v-if="visibleApps.length" :items="visibleApps" :item-height="48">
          <template #default="{ item }">
            <div class="grid h-12 grid-cols-[1fr_260px] items-center gap-3 border-b border-line px-3.5 text-xs hover:bg-inset">
              <div class="flex min-w-0 items-center gap-2.5">
                <div
                  class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-xs font-700"
                  :style="{
                    background: avatarStyle(item.package).bg,
                    color: avatarStyle(item.package).fg,
                  }"
                >
                  {{ avatarText(item) }}
                </div>
                <div class="min-w-0">
                  <div class="overflow-hidden text-ellipsis whitespace-nowrap text-xs font-600 leading-[1.4] text-fg">
                    {{ item.label || item.package }}
                  </div>
                  <div
                    v-if="item.label"
                    class="overflow-hidden text-ellipsis whitespace-nowrap font-mono text-[10px] leading-[1.4] text-fg3"
                  >
                    {{ item.package }}
                  </div>
                </div>
              </div>
              <div class="flex items-center justify-end gap-1.5">
                <NButton size="tiny" quaternary @click="openDetail(item.package)">{{ t('apps.detail') }}</NButton>
                <NButton size="tiny" quaternary @click="exportApp(item.package)">{{ t('apps.export') }}</NButton>
                <NPopconfirm @positive-click="uninstall(item.package)">
                  <template #trigger>
                    <NButton size="tiny" quaternary type="error">{{ t('apps.uninstall') }}</NButton>
                  </template>
                  {{ t('apps.uninstallConfirm', { pkg: item.package }) }}
                </NPopconfirm>
                <NDropdown :options="moreOptions" @select="(key) => onMore(String(key), item.package)">
                  <NButton size="tiny" quaternary>{{ t('apps.more') }}</NButton>
                </NDropdown>
              </div>
            </div>
          </template>
        </VirtualList>
        <div v-else class="py-10 text-center text-xs text-fg3">
          {{ store.current ? t('apps.noMatch') : t('apps.selectDeviceShort') }}
        </div>
      </div>
    </section>

    <NDrawer v-model:show="showDetail" :width="600">
      <NDrawerContent :title="t('apps.detailTitle')" closable :native-scrollbar="false">
        <NSpin :show="detailLoading">
          <div v-if="detail" class="flex flex-col">
            <div class="flex items-center gap-3.5 rounded border border-line bg-gradient-to-br from-inset to-surface p-4">
              <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-brand-soft text-xl font-700 text-brand-ink">
                {{ avatarLetter(detail) }}
              </div>
              <div class="min-w-0 flex-1">
                <div class="text-[15px] font-600">{{ detail.label || detail.package }}</div>
                <div class="mt-0.5 break-all font-mono text-[11px] text-fg3">{{ detail.package }}</div>
              </div>
              <div class="flex gap-2">
                <NButton size="small" @click="onMore('launch', detail.package)">{{ t('apps.launch') }}</NButton>
                <NButton size="small" @click="exportApp(detail.package)">{{ t('apps.export') }}</NButton>
                <NPopconfirm @positive-click="uninstallAndClose(detail.package)">
                  <template #trigger>
                    <NButton size="small" type="error" ghost>{{ t('apps.uninstall') }}</NButton>
                  </template>
                  {{ t('apps.uninstallConfirm', { pkg: detail.package }) }}
                </NPopconfirm>
              </div>
            </div>

            <div class="mt-3.5 grid grid-cols-2 gap-2.5">
              <div class="rounded-sm border border-line bg-surface px-3 py-2.5">
                <span class="block text-[10px] text-fg3">{{ t('apps.version') }}</span>
                <span class="mt-[3px] block break-all text-xs">{{ detail.versionName }} ({{ detail.versionCode }})</span>
              </div>
              <div class="rounded-sm border border-line bg-surface px-3 py-2.5">
                <span class="block text-[10px] text-fg3">SDK</span>
                <span class="mt-[3px] block break-all text-xs">min {{ detail.minSdk }} · target {{ detail.targetSdk }}</span>
              </div>
              <div class="rounded-sm border border-line bg-surface px-3 py-2.5">
                <span class="block text-[10px] text-fg3">{{ t('apps.firstInstall') }}</span>
                <span class="mt-[3px] block break-all text-xs">{{ detail.firstInstall || '—' }}</span>
              </div>
              <div class="rounded-sm border border-line bg-surface px-3 py-2.5">
                <span class="block text-[10px] text-fg3">{{ t('apps.lastUpdate') }}</span>
                <span class="mt-[3px] block break-all text-xs">{{ detail.lastUpdate || '—' }}</span>
              </div>
              <div class="col-span-2 rounded-sm border border-line bg-surface px-3 py-2.5">
                <span class="block text-[10px] text-fg3">{{ t('apps.dataDir') }}</span>
                <span class="mt-[3px] block break-all font-mono text-[11px]">{{ detail.dataDir || '—' }}</span>
              </div>
            </div>

            <div class="mt-4.5">
              <div class="mb-2 flex items-center justify-between text-xs font-600 text-fg">
                <span>
                  {{ t('apps.grantedPerms') }}
                  <em class="ml-1 font-400 not-italic text-fg3">{{ detail.permissions.length }}</em>
                </span>
                <NButton v-if="detail.permissions.length > 6" text size="tiny" @click="showAllPerms = !showAllPerms">
                  {{ showAllPerms ? t('apps.collapse') : t('apps.expandAll') }}
                </NButton>
              </div>
              <ul v-if="detail.permissions.length" class="m-0 list-none overflow-hidden rounded-sm border border-line p-0">
                <li
                  v-for="perm in visiblePerms"
                  :key="perm"
                  class="break-all border-b border-line px-2.5 py-1.5 font-mono text-[11px] last:border-b-0 even:bg-inset"
                >
                  {{ perm }}
                </li>
              </ul>
              <div v-else class="rounded-sm border border-dashed border-line p-4 text-center text-xs text-fg3">
                {{ t('apps.noPerms') }}
              </div>
            </div>
          </div>
          <div v-else-if="detailLoading" class="flex min-h-[240px] flex-col items-center justify-center gap-2.5 text-xs text-fg2">
            <div class="text-[13px]">{{ t('apps.loading') }}</div>
            <div class="h-1.5 w-[220px] overflow-hidden rounded-[3px] bg-line">
              <i class="block h-full bg-brand transition-[width] duration-200" :style="{ width: `${detailProgress}%` }"></i>
            </div>
            <div class="text-[11px] tabular-nums text-fg3">{{ detailProgress }}%</div>
          </div>
        </NSpin>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>
