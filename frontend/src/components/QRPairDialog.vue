<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { NButton, NModal, NSpin, useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import { CreatePairingQR, WaitQRPairing } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'

const props = defineProps<{ show: boolean }>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'paired'): void
}>()

const message = useMessage()

const qrData = ref('')
const status = ref('正在生成二维码…')
const waiting = ref(false)
const error = ref('')

let session: model.QRPairingSession | null = null
let runToken = 0

async function start() {
  const token = ++runToken
  error.value = ''
  waiting.value = true
  status.value = '正在生成二维码…'

  try {
    session = await CreatePairingQR()
    qrData.value = await QRCode.toDataURL(session.payload, {
      width: 280,
      margin: 1,
      errorCorrectionLevel: 'M',
    })
    status.value = '请用手机扫描二维码，等待手机广播配对服务…'

    const result = await WaitQRPairing(session.name, session.password)
    if (token !== runToken) {
      return
    }

    if (result.ok) {
      status.value = result.message
      message.success(result.message)
      emit('paired')
      emit('update:show', false)
    } else {
      error.value = result.message || '配对未完成'
      status.value = '配对未完成，可刷新二维码重试'
    }
  } catch (e) {
    if (token === runToken) {
      error.value = e instanceof Error ? e.message : String(e)
      status.value = '配对失败，可刷新二维码重试'
    }
  } finally {
    if (token === runToken) {
      waiting.value = false
    }
  }
}

function refresh() {
  void start()
}

function close() {
  runToken += 1
  waiting.value = false
  emit('update:show', false)
}

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      void start()
    } else {
      runToken += 1
    }
  },
)

onBeforeUnmount(() => {
  runToken += 1
})
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="扫码配对"
    style="width: 460px"
    :bordered="false"
    @update:show="(value: boolean) => (value ? null : close())"
  >
    <div class="qr">
      <div class="qr-box">
        <img v-if="qrData" :src="qrData" class="qr-image" alt="配对二维码" />
        <NSpin v-else size="small" />
      </div>

      <p class="status">{{ status }}</p>
      <p v-if="error" class="error">{{ error }}</p>

      <div class="actions">
        <NButton size="small" @click="close">取消</NButton>
        <NButton size="small" type="primary" @click="refresh">刷新二维码</NButton>
      </div>

      <p class="tip">
        手机端：设置 → 开发者选项 → 无线调试 → 「使用二维码配对设备」，用手机扫描上方二维码。
      </p>
    </div>
  </NModal>
</template>

<style scoped>
.qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.qr-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 300px;
  height: 300px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: #fff;
}

.qr-image {
  width: 280px;
  height: 280px;
}

.status {
  margin: 0;
  color: var(--text-secondary);
  font-size: 12px;
  text-align: center;
}

.error {
  margin: 0;
  color: var(--danger);
  font-size: 12px;
  text-align: center;
  word-break: break-all;
}

.actions {
  display: flex;
  gap: 8px;
}

.tip {
  margin: 0;
  color: var(--text-tertiary);
  font-size: 11px;
  line-height: 1.6;
  text-align: center;
}
</style>
