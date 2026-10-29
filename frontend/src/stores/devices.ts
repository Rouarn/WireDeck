import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { EnsureAdbReady, ListDevices, ReconnectDevices } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'

export const useDevicesStore = defineStore('devices', () => {
  const list = ref<model.Device[]>([])
  const adb = ref<model.AdbEnv | null>(null)
  const currentSerial = ref('')
  const error = ref('')
  const loading = ref(false)

  const current = computed(
    () => list.value.find((item) => item.serial === currentSerial.value) ?? null,
  )

  function apply(next: model.Device[]) {
    list.value = next
    if (!next.some((item) => item.serial === currentSerial.value)) {
      currentSerial.value = next[0]?.serial ?? ''
    }
  }

  async function refresh() {
    loading.value = true
    error.value = ''
    try {
      adb.value = await EnsureAdbReady()
      const devices = await ListDevices()
      apply(devices)
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  function select(serial: string) {
    currentSerial.value = serial
  }

  // reconnect 尝试重连历史设备并自动连接 mDNS 发现的无线设备。
  async function reconnect() {
    try {
      await ReconnectDevices()
    } catch {
      // 重连失败不打断页面，后续轮询会继续更新
    }
    await refresh()
  }

  return { list, adb, current, currentSerial, error, loading, apply, refresh, select, reconnect }
})
