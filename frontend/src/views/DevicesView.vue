<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { NAlert, NButton, NEmpty, NIcon, NInput, NPopconfirm, NSpin, NTag, NTooltip, useDialog, useMessage } from 'naive-ui'
import { AddOutline, InformationCircleOutline, PhonePortraitOutline } from '@vicons/ionicons5'
import { ConnectDevice, DisconnectDevice, DiscoverServices, EnableWireless, PairDevice, StartScrcpy } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import QRPairDialog from '../components/QRPairDialog.vue'
import { useDevicesStore } from '../stores/devices'
import { t } from '../i18n'

const store = useDevicesStore()
const message = useMessage()
const dialog = useDialog()

const host = ref('')
const pairPort = ref('')
const pairCode = ref('')
const connectPort = ref('')
const busy = ref(false)
const error = ref('')
const pairHint = ref('')
const services = ref<model.MDNSService[]>([])

// 已配对并连接的设备不再出现在「发现的服务」里。
const visibleServices = computed(() =>
  services.value.filter((svc) => {
    if (svc.kind !== 'connect') {
      return true
    }
    return !store.list.some(
      (device) =>
        device.serial === svc.name ||
        device.serial.includes(svc.name) ||
        svc.name.includes(device.serial),
    )
  }),
)
const showPairing = ref(false)
const pairingRef = ref<HTMLElement | null>(null)
const qrShow = ref(false)
const paired = ref(false)

async function onPaired() {
  await store.refresh()
}

function togglePairing() {
  showPairing.value = !showPairing.value
  if (showPairing.value) {
    // 只在需要配对时才做 mDNS 服务发现。
    void discover()
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
  if (state === 'device') return t('devices.state.connected')
  if (state === 'unauthorized') return t('devices.state.unauthorized')
  if (state === 'offline') return t('devices.state.offline')
  return state
}

function signalLabel(rssi: number): string {
  if (!rssi) {
    return '—'
  }
  if (rssi >= -55) {
    return t('devices.signal.strong')
  }
  if (rssi >= -70) {
    return t('devices.signal.medium')
  }
  return t('devices.signal.weak')
}

function lastSeenLabel(ts: number): string {
  if (!ts) {
    return '—'
  }
  const minutes = Math.floor((Date.now() - ts) / 60000)
  if (minutes < 1) {
    return t('devices.justNow')
  }
  if (minutes < 60) {
    return t('devices.minutesAgo', { n: minutes })
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return t('devices.hoursAgo', { n: hours })
  }
  return t('devices.daysAgo', { n: Math.floor(hours / 24) })
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
    error.value = t('devices.fillPairing')
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
      paired.value = true
      pairHint.value = t('devices.pairingOk')
      message.success(t('devices.pairedOk'))
    } else {
      error.value = res.message || t('devices.pairingFail')
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
    error.value = t('devices.fillConnect')
    return
  }

  busy.value = true
  try {
    const res = await ConnectDevice({ host: host.value.trim(), port: Number(connectPort.value) })
    if (res.ok) {
      pairHint.value = ''
      message.success(t('devices.connected'))
      await store.refresh()
    } else {
      error.value = res.message || t('devices.connectFail')
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
    message.info(t('devices.disconnected'))
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

async function enableWireless(serial: string) {
  busy.value = true
  try {
    const res = await EnableWireless(serial)
    if (res.ok) {
      message.success(res.message || t('devices.wirelessOk'))
      await store.refresh()
    } else {
      message.warning(res.message || t('devices.wirelessFail'))
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    busy.value = false
  }
}

async function doStartScrcpy(device: model.Device) {
  try {
    await StartScrcpy(device.serial, device.model || device.serial)
    message.success(t('devices.mirrorStarted'))
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function startScrcpy(device: model.Device) {
  // 小米 / 红米默认关闭「USB 调试（安全设置）」，投屏能看但不能控制；首次提示一次。
  const isXiaomi = /xiaomi|redmi|poco/i.test(device.brand || '')
  const tipKey = 'wiredeck.scrcpy.xiaomi-tip'

  if (isXiaomi && !localStorage.getItem(tipKey)) {
    localStorage.setItem(tipKey, '1')
    dialog.info({
      title: t('devices.tipTitle'),
      content: t('devices.tipContent'),
      positiveText: t('devices.tipContinue'),
      onPositiveClick: () => {
        void doStartScrcpy(device)
      },
    })
    return
  }

  await doStartScrcpy(device)
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

onMounted(() => {
  // 设备列表由后台轮询维护，进入页面不重复重连；
  // 启动时的自动重连已在后端 startup 中完成。
  if (!store.list.length) {
    void store.refresh()
  }
})
</script>

<template>
  <div class="devices">
    <NAlert v-if="error" type="error" closable class="alert" @close="error = ''">
      {{ error }}
    </NAlert>

    <div v-if="devices.length" class="notice">
      <NIcon :size="14"><InformationCircleOutline /></NIcon>
      <span>{{ t('devices.detected', { n: devices.length }) }}</span>
    </div>

    <div class="section-head">
      <h2>{{ t('devices.connected') }}</h2>
      <span class="count">{{ t('devices.count', { n: devices.length }) }}</span>
    </div>

    <NSpin :show="store.loading">
      <div class="device-grid">
        <article
          v-for="device in devices"
          :key="device.serial"
          class="device-card"
          :class="{ current: store.current?.serial === device.serial }"
        >
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
              <span class="meta-label">{{ t('devices.serial') }}</span>
              <span class="meta-value"><code :title="device.serial">{{ device.serial }}</code></span>
            </div>
            <div>
              <span class="meta-label">{{ t('devices.ip') }}</span>
              <span class="meta-value">{{ device.ip || '—' }}</span>
            </div>
            <div>
              <span class="meta-label">{{ t('devices.signal') }}</span>
              <span class="meta-value">
                {{ device.transport === 'wifi' ? signalLabel(device.rssi) : '—' }}
              </span>
            </div>
            <div>
              <span class="meta-label">{{ t('devices.status') }}</span>
              <span class="meta-value">
                {{ stateLabel(device.state) }}
                <template v-if="device.state !== 'device'"> · {{ lastSeenLabel(device.lastSeen) }}</template>
              </span>
            </div>
          </div>

          <div class="card-actions">
            <span v-if="store.current?.serial === device.serial" class="current-tag">{{ t('devices.current') }}</span>
            <NButton v-else size="small" type="primary" @click="store.select(device.serial)">{{ t('devices.setCurrent') }}</NButton>
            <NTooltip>
              <template #trigger>
                <NButton
                  size="small"
                  quaternary
                  :disabled="device.state !== 'device'"
                  @click="startScrcpy(device)"
                >
                  {{ t('devices.mirror') }}
                </NButton>
              </template>
              {{ t('devices.mirrorTip') }}
            </NTooltip>
            <NButton
              v-if="device.transport === 'usb'"
              size="small"
              quaternary
              :loading="busy"
              :disabled="device.state !== 'device'"
              @click="enableWireless(device.serial)"
            >
              {{ t('devices.wireless') }}
            </NButton>
            <NPopconfirm v-if="device.transport === 'wifi'" @positive-click="disconnect(device.serial)">
              <template #trigger>
                <NButton size="small" quaternary>{{ t('devices.disconnect') }}</NButton>
              </template>
              {{ t('devices.disconnectConfirm', { name: device.model || device.serial }) }}
            </NPopconfirm>
          </div>
        </article>

        <button class="add-card" @click="togglePairing">
          <NIcon :size="18"><AddOutline /></NIcon>
          <strong>{{ t('devices.add') }}</strong>
          <span>{{ t('devices.addHint') }}</span>
        </button>
      </div>

      <NEmpty
        v-if="!devices.length && !store.loading"
        class="empty"
        :description="t('devices.empty')"
      />
    </NSpin>

    <section v-if="showPairing" ref="pairingRef" class="pairing">
      <div class="pairing-head">
        <div>
          <h3>{{ t('devices.pairTitle') }}</h3>
          <p class="pairing-sub">{{ t('devices.pairSub') }}</p>
        </div>
        <NButton size="small" type="primary" @click="qrShow = true">{{ t('devices.qrPair') }}</NButton>
      </div>

      <div class="pairing-body">
        <div v-if="visibleServices.length" class="services">
          <span class="services-label">{{ t('devices.services') }}</span>
          <button
            v-for="service in visibleServices"
            :key="service.name + service.kind"
            class="service-chip"
            @click="fillFromService(service)"
          >
            {{ service.kind === 'pairing' ? t('devices.kindPair') : service.kind === 'connect' ? t('devices.kindConnect') : t('devices.kindOther') }}
            · {{ service.address }}
          </button>
        </div>

        <div class="pair-step" :class="{ done: paired }">
          <div class="step-head">
            <span class="step-no">{{ paired ? '✓' : '1' }}</span>
            <span class="step-title">{{ t('devices.stepPair') }}</span>
            <span class="step-hint">{{ t('devices.stepPairHint') }}</span>
          </div>
          <div class="pair-fields">
            <label class="field">
              <span>{{ t('devices.ip') }}</span>
              <NInput v-model:value="host" size="small" placeholder="192.168.1.23" />
            </label>
            <label class="field">
              <span>{{ t('devices.pairPort') }}</span>
              <NInput v-model:value="pairPort" size="small" placeholder="37123" />
            </label>
            <label class="field">
              <span>{{ t('devices.pairCode') }}</span>
              <NInput v-model:value="pairCode" size="small" :placeholder="t('devices.pairCodePlaceholder')" maxlength="6" />
            </label>
          </div>
          <div class="step-actions">
            <NButton size="small" type="primary" :loading="busy" @click="pair">{{ t('devices.pairBtn') }}</NButton>
          </div>
        </div>

        <div class="pair-step">
          <div class="step-head">
            <span class="step-no">2</span>
            <span class="step-title">{{ t('devices.stepConnect') }}</span>
            <span class="step-hint">{{ t('devices.connectHint') }}</span>
          </div>
          <div class="pair-fields single">
            <label class="field">
              <span>{{ t('devices.connectPort') }}</span>
              <NInput v-model:value="connectPort" size="small" placeholder="40123" />
            </label>
          </div>
          <div class="step-actions">
            <NButton size="small" :loading="busy" @click="connect">{{ t('devices.connectBtn') }}</NButton>
            <NButton size="small" quaternary :loading="busy" @click="discover">{{ t('devices.rediscover') }}</NButton>
          </div>
        </div>

        <p v-if="pairHint" class="pair-hint">{{ pairHint }}</p>
      </div>
    </section>
    <QRPairDialog v-model:show="qrShow" @paired="onPaired" />
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

.device-card.current {
  border-color: var(--primary-border);
  box-shadow: 0 0 0 1px var(--primary-border), var(--shadow-sm);
}

.current-tag {
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  border-radius: 6px;
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 11px;
  font-weight: 600;
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
  grid-template-columns: repeat(4, minmax(0, 1fr));
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

.pairing-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
}

.pairing-head h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.pairing-sub {
  margin: 3px 0 0;
  color: var(--text-tertiary);
  font-size: 11px;
}

.pairing-head .n-button {
  margin-left: auto;
}

.pairing-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
}

.pair-step {
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-inset);
}

.pair-step.done {
  border-color: var(--success);
}

.step-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.step-no {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--primary-soft);
  color: var(--primary-ink);
  font-size: 10px;
  font-weight: 700;
}

.pair-step.done .step-no {
  background: var(--success-soft);
  color: var(--success);
}

.step-title {
  font-size: 12px;
  font-weight: 600;
}

.step-hint {
  margin-left: auto;
  color: var(--text-tertiary);
  font-size: 10px;
}

.pair-fields {
  display: grid;
  grid-template-columns: 1.4fr 1fr 1fr;
  gap: 10px;
}

.pair-fields.single {
  grid-template-columns: 1fr;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field > span {
  color: var(--text-tertiary);
  font-size: 10px;
}

.step-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.services {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
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

.pair-hint {
  margin: 10px 0 0;
  color: var(--success);
  font-size: 12px;
}
</style>
