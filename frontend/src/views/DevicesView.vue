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
  <div class="flex flex-col">
    <NAlert v-if="error" type="error" closable class="mb-4" @close="error = ''">
      {{ error }}
    </NAlert>

    <div
      v-if="devices.length"
      class="mb-4 flex items-center gap-2 rounded-sm border border-brand-line bg-brand-soft px-3 py-2.5 text-xs text-brand-ink"
    >
      <NIcon :size="14"><InformationCircleOutline /></NIcon>
      <span>{{ t('devices.detected', { n: devices.length }) }}</span>
    </div>

    <div class="mb-2.5 mt-1 flex items-baseline gap-2">
      <h2 class="m-0 text-[13px] font-600">{{ t('devices.connected') }}</h2>
      <span class="text-xs text-fg3">{{ t('devices.count', { n: devices.length }) }}</span>
    </div>

    <NSpin :show="store.loading">
      <div class="grid grid-cols-2 gap-3">
        <article
          v-for="device in devices"
          :key="device.serial"
          class="rounded border bg-surface p-4 shadow-sm"
          :class="
            store.current?.serial === device.serial
              ? 'border-brand-line shadow-[0_0_0_1px_var(--primary-border),var(--shadow-sm)]'
              : 'border-line'
          "
        >
          <div class="flex items-center gap-3">
            <div class="flex h-[38px] w-[38px] items-center justify-center rounded-[10px] bg-brand-soft text-brand-ink">
              <NIcon :size="20"><PhonePortraitOutline /></NIcon>
            </div>
            <div>
              <div class="text-[13px] font-600">{{ device.model || device.serial }}</div>
              <div class="mt-px text-[11px] text-fg3">
                {{ device.transport === 'wifi' ? 'Wi-Fi' : 'USB' }} · {{ device.state }}
              </div>
            </div>
            <NTag class="ml-auto" size="small" :type="tagType(device.state)" :bordered="false">
              {{ stateLabel(device.state) }}
            </NTag>
          </div>

          <div class="my-3.5 grid grid-cols-4 gap-2 rounded-sm bg-inset px-3 py-2.5">
            <div>
              <span class="block text-[10px] text-fg3">{{ t('devices.serial') }}</span>
              <span class="mt-0.5 block overflow-hidden text-ellipsis whitespace-nowrap text-xs text-fg">
                <code :title="device.serial" class="text-[11px]">{{ device.serial }}</code>
              </span>
            </div>
            <div>
              <span class="block text-[10px] text-fg3">{{ t('devices.ip') }}</span>
              <span class="mt-0.5 block overflow-hidden text-ellipsis whitespace-nowrap text-xs text-fg">
                {{ device.ip || '—' }}
              </span>
            </div>
            <div>
              <span class="block text-[10px] text-fg3">{{ t('devices.signal') }}</span>
              <span class="mt-0.5 block overflow-hidden text-ellipsis whitespace-nowrap text-xs text-fg">
                {{ device.transport === 'wifi' ? signalLabel(device.rssi) : '—' }}
              </span>
            </div>
            <div>
              <span class="block text-[10px] text-fg3">{{ t('devices.status') }}</span>
              <span class="mt-0.5 block overflow-hidden text-ellipsis whitespace-nowrap text-xs text-fg">
                {{ stateLabel(device.state) }}
                <template v-if="device.state !== 'device'"> · {{ lastSeenLabel(device.lastSeen) }}</template>
              </span>
            </div>
          </div>

          <div class="flex gap-2">
            <span
              v-if="store.current?.serial === device.serial"
              class="inline-flex items-center rounded-[6px] bg-brand-soft px-2 text-[11px] font-600 text-brand-ink"
            >
              {{ t('devices.current') }}
            </span>
            <NButton v-else size="small" type="primary" @click="store.select(device.serial)">
              {{ t('devices.setCurrent') }}
            </NButton>
            <NTooltip>
              <template #trigger>
                <NButton size="small" quaternary :disabled="device.state !== 'device'" @click="startScrcpy(device)">
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

        <button
          class="flex min-h-[168px] cursor-pointer flex-col items-center justify-center gap-1.5 rounded border border-dashed border-line-strong bg-transparent text-xs text-fg3 hover:border-brand-line hover:text-brand-ink"
          @click="togglePairing"
        >
          <NIcon :size="18"><AddOutline /></NIcon>
          <strong class="font-500 text-fg2">{{ t('devices.add') }}</strong>
          <span>{{ t('devices.addHint') }}</span>
        </button>
      </div>

      <NEmpty v-if="!devices.length && !store.loading" class="my-3" :description="t('devices.empty')" />
    </NSpin>

    <section v-if="showPairing" ref="pairingRef" class="mt-5 rounded border border-line bg-surface shadow-sm">
      <div class="flex items-center gap-3 border-b border-line px-4 py-3.5">
        <div>
          <h3 class="m-0 text-[13px] font-600">{{ t('devices.pairTitle') }}</h3>
          <p class="mb-0 mt-[3px] text-[11px] text-fg3">{{ t('devices.pairSub') }}</p>
        </div>
        <NButton class="ml-auto" size="small" type="primary" @click="qrShow = true">{{ t('devices.qrPair') }}</NButton>
      </div>

      <div class="flex flex-col gap-3 p-4">
        <div v-if="visibleServices.length" class="flex flex-wrap items-center gap-2">
          <span class="text-[11px] text-fg3">{{ t('devices.services') }}</span>
          <button
            v-for="service in visibleServices"
            :key="service.name + service.kind"
            class="cursor-pointer rounded-full border border-brand-line bg-brand-soft px-2.5 py-1 text-[11px] text-brand-ink"
            @click="fillFromService(service)"
          >
            {{ service.kind === 'pairing' ? t('devices.kindPair') : service.kind === 'connect' ? t('devices.kindConnect') : t('devices.kindOther') }}
            · {{ service.address }}
          </button>
        </div>

        <div
          class="rounded-sm border bg-inset px-3.5 py-3"
          :class="paired ? 'border-ok' : 'border-line'"
        >
          <div class="mb-2.5 flex items-center gap-2">
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-700"
              :class="paired ? 'bg-[var(--success-soft)] text-ok' : 'bg-brand-soft text-brand-ink'"
            >
              {{ paired ? '✓' : '1' }}
            </span>
            <span class="text-xs font-600">{{ t('devices.stepPair') }}</span>
            <span class="ml-auto text-[10px] text-fg3">{{ t('devices.stepPairHint') }}</span>
          </div>
          <div class="grid grid-cols-[1.4fr_1fr_1fr] gap-2.5">
            <label class="flex flex-col gap-1">
              <span class="text-[10px] text-fg3">{{ t('devices.ip') }}</span>
              <NInput v-model:value="host" size="small" placeholder="192.168.1.23" />
            </label>
            <label class="flex flex-col gap-1">
              <span class="text-[10px] text-fg3">{{ t('devices.pairPort') }}</span>
              <NInput v-model:value="pairPort" size="small" placeholder="37123" />
            </label>
            <label class="flex flex-col gap-1">
              <span class="text-[10px] text-fg3">{{ t('devices.pairCode') }}</span>
              <NInput v-model:value="pairCode" size="small" :placeholder="t('devices.pairCodePlaceholder')" maxlength="6" />
            </label>
          </div>
          <div class="mt-2.5 flex justify-end gap-2">
            <NButton size="small" type="primary" :loading="busy" @click="pair">{{ t('devices.pairBtn') }}</NButton>
          </div>
        </div>

        <div class="rounded-sm border border-line bg-inset px-3.5 py-3">
          <div class="mb-2.5 flex items-center gap-2">
            <span class="flex h-5 w-5 items-center justify-center rounded-full bg-brand-soft text-[10px] font-700 text-brand-ink">
              2
            </span>
            <span class="text-xs font-600">{{ t('devices.stepConnect') }}</span>
            <span class="ml-auto text-[10px] text-fg3">{{ t('devices.connectHint') }}</span>
          </div>
          <div class="grid grid-cols-1 gap-2.5">
            <label class="flex flex-col gap-1">
              <span class="text-[10px] text-fg3">{{ t('devices.connectPort') }}</span>
              <NInput v-model:value="connectPort" size="small" placeholder="40123" />
            </label>
          </div>
          <div class="mt-2.5 flex justify-end gap-2">
            <NButton size="small" :loading="busy" @click="connect">{{ t('devices.connectBtn') }}</NButton>
            <NButton size="small" quaternary :loading="busy" @click="discover">{{ t('devices.rediscover') }}</NButton>
          </div>
        </div>

        <p v-if="pairHint" class="mt-2.5 text-xs text-ok">{{ pairHint }}</p>
      </div>
    </section>
    <QRPairDialog v-model:show="qrShow" @paired="onPaired" />
  </div>
</template>
