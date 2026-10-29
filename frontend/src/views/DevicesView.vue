<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { NAlert, NButton, NEmpty, NIcon, NInput, NSpin, NTag, useMessage } from 'naive-ui'
import { AddOutline, InformationCircleOutline, PhonePortraitOutline } from '@vicons/ionicons5'
import { ConnectDevice, DisconnectDevice, DiscoverServices, PairDevice } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { useDevicesStore } from '../stores/devices'

const store = useDevicesStore()
const message = useMessage()

const host = ref('')
const pairPort = ref('')
const pairCode = ref('')
const connectPort = ref('')
const busy = ref(false)
const error = ref('')
const pairHint = ref('')
const services = ref<model.MDNSService[]>([])
const showPairing = ref(false)
const pairingRef = ref<HTMLElement | null>(null)

function togglePairing() {
  showPairing.value = !showPairing.value
  if (showPairing.value) {
    void nextTick(() => pairingRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
  }
}

const devices = computed(() => store.list)

function tagType(state: string): 'success' | 'warning' | 'error' {
  if (state === 'device') return 'success'
  if (state === 'unauthorized') return 'warning'
  return 'error'
}

function stateLabel(state: string): string {
  if (state === 'device') return '已连接'
  if (state === 'unauthorized') return '待授权'
  if (state === 'offline') return '离线'
  return state
}

async function discover() {
  try {
    services.value = await DiscoverServices()
  } catch {
    services.value = []
  }
}

async function pair() {
  error.value = ''
  pairHint.value = ''
  if (!host.value.trim() || !pairPort.value || !pairCode.value) {
    error.value = '请填写 IP、配对端口和配对码'
    return
  }

  busy.value = true
  try {
    const res = await PairDevice({
      host: host.value.trim(),
      pairPort: Number(pairPort.value),
      code: pairCode.value.trim(),
    })
    if (res.ok) {
      pairHint.value = '配对成功。现在填写无线调试主页显示的「IP 地址和端口」中的端口，点击连接。'
      message.success('配对成功')
    } else {
      error.value = res.message || '配对失败'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

async function connect() {
  error.value = ''
  if (!host.value.trim() || !connectPort.value) {
    error.value = '请填写 IP 和连接端口'
    return
  }

  busy.value = true
  try {
    const res = await ConnectDevice({ host: host.value.trim(), port: Number(connectPort.value) })
    if (res.ok) {
      pairHint.value = ''
      message.success('已连接')
      await store.refresh()
    } else {
      error.value = res.message || '连接失败'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

async function disconnect(serial: string) {
  try {
    await DisconnectDevice(serial)
    await store.refresh()
    message.info('已断开')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

function fillFromService(service: model.MDNSService) {
  host.value = service.host
  if (service.kind === 'pairing') {
    pairPort.value = String(service.port)
  }
  if (service.kind === 'connect') {
    connectPort.value = String(service.port)
  }
}

onMounted(async () => {
  await store.reconnect()
  await discover()
})
</script>

<template>
  <div class="devices">
    <NAlert v-if="error" type="error" closable class="alert" @close="error = ''">
      {{ error }}
    </NAlert>

    <div v-if="devices.length" class="notice">
      <NIcon :size="14"><InformationCircleOutline /></NIcon>
      <span>已检测到 {{ devices.length }} 台设备，可直接安装或管理。</span>
    </div>

    <div class="section-head">
      <h2>已连接</h2>
      <span class="count">{{ devices.length }} 台</span>
    </div>

    <NSpin :show="store.loading">
      <div class="device-grid">
        <article v-for="device in devices" :key="device.serial" class="device-card">
          <div class="card-head">
            <div class="device-avatar">
              <NIcon :size="20"><PhonePortraitOutline /></NIcon>
            </div>
            <div>
              <div class="device-title">{{ device.model || device.serial }}</div>
              <div class="device-sub">
                {{ device.transport === 'wifi' ? 'Wi-Fi' : 'USB' }} · {{ device.state }}
              </div>
            </div>
            <NTag class="state-tag" size="small" :type="tagType(device.state)" :bordered="false">
              {{ stateLabel(device.state) }}
            </NTag>
          </div>

          <div class="meta-grid">
            <div>
              <span class="meta-label">序列号</span>
              <span class="meta-value"><code>{{ device.serial }}</code></span>
            </div>
            <div>
              <span class="meta-label">传输方式</span>
              <span class="meta-value">{{ device.transport === 'wifi' ? 'Wi-Fi' : 'USB' }}</span>
            </div>
            <div>
              <span class="meta-label">状态</span>
              <span class="meta-value">{{ stateLabel(device.state) }}</span>
            </div>
          </div>

          <div class="card-actions">
            <NButton size="small" type="primary" @click="store.select(device.serial)">设为当前</NButton>
            <NButton size="small" quaternary @click="disconnect(device.serial)">断开</NButton>
          </div>
        </article>

        <button class="add-card" @click="togglePairing">
          <NIcon :size="18"><AddOutline /></NIcon>
          <strong>添加设备</strong>
          <span>无线配对或插入数据线</span>
        </button>
      </div>

      <NEmpty
        v-if="!devices.length && !store.loading"
        class="empty"
        description="暂无设备。用数据线连接手机，或在手机上开启无线调试后配对。"
      />
    </NSpin>

    <section v-if="showPairing" ref="pairingRef" class="pairing">
      <div class="panel-head">
        <h3>无线调试配对</h3>
        <span class="hint">Android 11+</span>
      </div>
      <div class="panel-body">
        <ol class="steps">
          <li>手机打开 设置 → 开发者选项 → 无线调试，并保持页面开启。</li>
          <li>点击「使用配对码配对设备」，记下页面上的 IP、配对端口和 6 位配对码。</li>
          <li>先配对，再用「IP 地址和端口」中的连接端口建立连接。</li>
        </ol>

        <div v-if="services.length" class="services">
          <span class="services-label">发现的服务</span>
          <button
            v-for="service in services"
            :key="service.name + service.kind"
            class="service-chip"
            @click="fillFromService(service)"
          >
            {{ service.kind === 'pairing' ? '配对' : service.kind === 'connect' ? '连接' : '其他' }}
            · {{ service.address }}
          </button>
        </div>

        <div class="pair-form">
          <NInput v-model:value="host" size="small" placeholder="IP 地址" style="width: 150px" />
          <NInput v-model:value="pairPort" size="small" placeholder="配对端口" style="width: 120px" />
          <NInput
            v-model:value="pairCode"
            size="small"
            placeholder="配对码"
            maxlength="6"
            style="width: 120px"
          />
          <NButton size="small" :loading="busy" @click="pair">配对</NButton>
        </div>

        <div class="pair-form">
          <NInput
            v-model:value="connectPort"
            size="small"
            placeholder="连接端口"
            style="width: 120px"
          />
          <NButton size="small" type="primary" :loading="busy" @click="connect">连接</NButton>
          <NButton size="small" quaternary :loading="busy" @click="discover">重新发现</NButton>
        </div>

        <p v-if="pairHint" class="pair-hint">{{ pairHint }}</p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.devices {
  display: flex;
  flex-direction: column;
}

.alert {
  margin-bottom: 16px;
}

.notice {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 10px 12px;
  border: 1px solid var(--primary-border);
  border-radius: var(--radius-sm);
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 12px;
}

.section-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin: 4px 0 10px;
}

.section-head h2 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.section-head .count {
  color: var(--text-tertiary);
  font-size: 12px;
}

.device-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.device-card {
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.card-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.device-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: var(--primary-soft);
  color: var(--primary-ink);
}

.device-title {
  font-size: 13px;
  font-weight: 600;
}

.device-sub {
  margin-top: 1px;
  color: var(--text-tertiary);
  font-size: 11px;
}

.state-tag {
  margin-left: auto;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin: 14px 0;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  background: var(--surface-inset);
}

.meta-label {
  display: block;
  color: var(--text-tertiary);
  font-size: 10px;
}

.meta-value {
  display: block;
  margin-top: 2px;
  overflow: hidden;
  color: var(--text);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-value code {
  color: inherit;
  font-size: 11px;
}

.card-actions {
  display: flex;
  gap: 8px;
}

.add-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 168px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius);
  background: transparent;
  color: var(--text-tertiary);
  font-size: 12px;
  cursor: pointer;
}

.add-card:hover {
  border-color: var(--primary-border);
  color: var(--primary-ink);
}

.add-card strong {
  color: var(--text-secondary);
  font-weight: 500;
}

.empty {
  margin: 12px 0;
}

.pairing {
  margin-top: 20px;
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

.panel-body {
  padding: 16px;
}

.steps {
  display: grid;
  gap: 8px;
  margin: 0 0 14px;
  padding: 0;
  list-style: none;
  counter-reset: step;
}

.steps li {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  color: var(--text-secondary);
  font-size: 12px;
}

.steps li::before {
  counter-increment: step;
  content: counter(step);
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 18px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 10px;
  font-weight: 600;
}

.services {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.services-label {
  color: var(--text-tertiary);
  font-size: 11px;
}

.service-chip {
  padding: 4px 10px;
  border: 1px solid var(--primary-border);
  border-radius: 999px;
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 11px;
  cursor: pointer;
}

.pair-form {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}

.pair-hint {
  margin: 10px 0 0;
  color: var(--success);
  font-size: 12px;
}
</style>
