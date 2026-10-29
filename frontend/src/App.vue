<script lang="ts" setup>
import { onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NEmpty,
  NSpin,
  NTag,
} from 'naive-ui'
import { EnsureAdbReady, ListDevices } from '../wailsjs/go/main/App'
import type { model } from '../wailsjs/go/models'

const env = ref<model.AdbEnv | null>(null)
const devices = ref<model.Device[]>([])
const loading = ref(false)
const error = ref('')

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    env.value = await EnsureAdbReady()
    devices.value = await ListDevices()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

function stateType(state: string) {
  if (state === 'device') return 'success'
  if (state === 'unauthorized') return 'warning'
  return 'error'
}

onMounted(refresh)
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">
        <span class="logo">W</span>
        <span>WireDeck</span>
      </div>
      <div class="subtitle">无线 ADB 桌面控制台</div>
      <n-button size="small" :loading="loading" @click="refresh">重新检测</n-button>
    </header>

    <main class="content">
      <n-alert
        v-if="error"
        type="error"
        :title="error"
        closable
        class="alert"
        @close="error = ''"
      />

      <n-card title="adb 运行环境" size="small" class="card">
        <n-spin :show="loading">
          <n-descriptions v-if="env" :column="1" label-placement="left" size="small">
            <n-descriptions-item label="状态">
              <n-tag type="success" size="small">已就绪</n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="adb 路径">
              <code class="mono">{{ env.binaryPath }}</code>
            </n-descriptions-item>
            <n-descriptions-item label="版本">{{ env.version }}</n-descriptions-item>
            <n-descriptions-item label="server 端口">{{ env.serverPort }}</n-descriptions-item>
          </n-descriptions>
          <n-empty v-else description="尚未检测到 adb" />
        </n-spin>
      </n-card>

      <n-card title="设备" size="small" class="card">
        <n-spin :show="loading">
          <n-empty
            v-if="devices.length === 0"
            description="暂无设备。请用数据线连接手机，或在手机上开启无线调试。"
          />
          <div v-else class="device-list">
            <div v-for="d in devices" :key="d.serial" class="device-row">
              <div class="device-main">
                <div class="device-name">{{ d.model || d.serial }}</div>
                <div class="device-serial">{{ d.serial }} · {{ d.transport }}</div>
              </div>
              <n-tag :type="stateType(d.state)" size="small">{{ d.state }}</n-tag>
            </div>
          </div>
        </n-spin>
      </n-card>
    </main>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  background: #f7f8fa;
  color: #1f2937;
}

.topbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  background: #ffffff;
  border-bottom: 1px solid #e5e7eb;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
}

.logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 7px;
  background: #2563eb;
  color: #ffffff;
  font-size: 14px;
}

.subtitle {
  flex: 1;
  color: #6b7280;
  font-size: 12px;
}

.content {
  max-width: 880px;
  margin: 0 auto;
  padding: 20px;
}

.card {
  margin-bottom: 16px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  color: #374151;
}

.device-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.device-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.device-name {
  font-weight: 600;
}

.device-serial {
  margin-top: 2px;
  color: #6b7280;
  font-size: 12px;
}
</style>
