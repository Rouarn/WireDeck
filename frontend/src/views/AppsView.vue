<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NDescriptions,
  NDescriptionsItem,
  NDrawer,
  NDrawerContent,
  NDropdown,
  NInput,
  NPopconfirm,
  NRadioButton,
  NRadioGroup,
  NSpin,
  useMessage,
} from 'naive-ui'
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
import { useDevicesStore } from '../stores/devices'

const store = useDevicesStore()
const message = useMessage()

const apps = ref<model.AppSummary[]>([])
const loading = ref(false)
const search = ref('')
const scope = ref<'user' | 'system' | 'all'>('user')
const showDetail = ref(false)
const detailLoading = ref(false)
const detail = ref<model.AppDetail | null>(null)

const visibleApps = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) {
    return apps.value
  }
  return apps.value.filter((app) => app.package.toLowerCase().includes(keyword))
})

const moreOptions = [
  { label: '启动应用', key: 'launch' },
  { label: '强制停止', key: 'stop' },
  { label: '清除数据', key: 'clear' },
]

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
  try {
    detail.value = await fetchAppDetail(store.current.serial, pkg)
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
    await ExportApp(store.current.serial, pkg, dir)
    message.success(`已导出到 ${dir}`)
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
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

async function onMore(key: string, pkg: string) {
  if (!store.current) {
    return
  }
  try {
    let result: model.AppOpResult
    if (key === 'launch') {
      result = await LaunchApp(store.current.serial, pkg)
    } else if (key === 'stop') {
      result = await ForceStopApp(store.current.serial, pkg)
    } else {
      result = await ClearAppData(store.current.serial, pkg)
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

onMounted(load)
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
        placeholder="搜索包名"
        clearable
        style="width: 220px"
      />
      <NRadioGroup v-model:value="scope" size="small" @update:value="load">
        <NRadioButton value="user">第三方</NRadioButton>
        <NRadioButton value="system">系统</NRadioButton>
        <NRadioButton value="all">全部</NRadioButton>
      </NRadioGroup>
      <span class="count">{{ visibleApps.length }} / {{ apps.length }}</span>
      <div class="spacer"></div>
      <NButton size="small" :loading="loading" @click="load">刷新</NButton>
    </div>

    <section class="panel">
      <NSpin :show="loading">
        <table v-if="visibleApps.length" class="table">
          <thead>
            <tr>
              <th>包名</th>
              <th class="ops-col">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="app in visibleApps" :key="app.package">
              <td><code>{{ app.package }}</code></td>
              <td class="ops">
                <NButton size="tiny" quaternary @click="openDetail(app.package)">详情</NButton>
                <NButton size="tiny" quaternary @click="exportApp(app.package)">导出</NButton>
                <NPopconfirm @positive-click="uninstall(app.package)">
                  <template #trigger>
                    <NButton size="tiny" quaternary type="error">卸载</NButton>
                  </template>
                  确认卸载 {{ app.package }}？此操作不可撤销。
                </NPopconfirm>
                <NDropdown :options="moreOptions" @select="(key) => onMore(String(key), app.package)">
                  <NButton size="tiny" quaternary>更多</NButton>
                </NDropdown>
              </td>
            </tr>
          </tbody>
        </table>
        <div v-else class="empty">
          {{ store.current ? '没有匹配的应用' : '请先选择设备' }}
        </div>
      </NSpin>
    </section>

    <NDrawer v-model:show="showDetail" :width="440">
      <NDrawerContent title="应用详情" closable>
        <NSpin :show="detailLoading">
          <NDescriptions v-if="detail" :column="1" label-placement="left" size="small" bordered>
            <NDescriptionsItem label="包名">
              <code>{{ detail.package }}</code>
            </NDescriptionsItem>
            <NDescriptionsItem label="版本">{{ detail.versionName }} ({{ detail.versionCode }})</NDescriptionsItem>
            <NDescriptionsItem label="SDK">min {{ detail.minSdk }} / target {{ detail.targetSdk }}</NDescriptionsItem>
            <NDescriptionsItem label="首次安装">{{ detail.firstInstall || '—' }}</NDescriptionsItem>
            <NDescriptionsItem label="最近更新">{{ detail.lastUpdate || '—' }}</NDescriptionsItem>
            <NDescriptionsItem label="数据目录">{{ detail.dataDir || '—' }}</NDescriptionsItem>
            <NDescriptionsItem label="已授予权限">
              {{ detail.permissions.length ? detail.permissions.join('、') : '—' }}
            </NDescriptionsItem>
          </NDescriptions>
        </NSpin>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.apps {
  display: flex;
  flex-direction: column;
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

.spacer {
  flex: 1;
}

.panel {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
}

.table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.table th {
  padding: 8px 14px;
  border-bottom: 1px solid var(--border);
  color: var(--text-tertiary);
  font-size: 10px;
  font-weight: 500;
  text-align: left;
  text-transform: uppercase;
}

.table th.ops-col {
  width: 300px;
}

.table td {
  padding: 8px 14px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
}

.table tbody tr:hover {
  background: var(--surface-inset);
}

.table tr:last-child td {
  border-bottom: none;
}

.table code {
  color: var(--text);
  font-size: 11px;
}

.ops {
  display: flex;
  align-items: center;
  gap: 2px;
}

.empty {
  padding: 40px 0;
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}
</style>
