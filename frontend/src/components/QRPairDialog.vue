<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { NButton, NModal, useMessage } from 'naive-ui'
import jsQR from 'jsqr'
import { PairByQRCode } from '../../wailsjs/go/main/App'

const props = defineProps<{ show: boolean }>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'paired'): void
}>()

const message = useMessage()

const video = ref<HTMLVideoElement | null>(null)
const scanning = ref(false)
const pairing = ref(false)
const status = ref('点击「开始扫描」打开摄像头')
const error = ref('')

let stream: MediaStream | null = null
let frame = 0
let stopped = false

async function start() {
  error.value = ''
  status.value = '正在打开摄像头…'
  stopped = false

  try {
    stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment' },
      audio: false,
    })
    if (video.value) {
      video.value.srcObject = stream
      await video.value.play()
    }
    scanning.value = true
    status.value = '将手机上的配对二维码对准摄像头'
    scanLoop()
  } catch (e) {
    error.value = `无法打开摄像头：${e instanceof Error ? e.message : String(e)}`
    scanning.value = false
  }
}

function scanLoop() {
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d', { willReadFrequently: true })

  const tick = () => {
    if (stopped) {
      return
    }
    const el = video.value
    if (el && ctx && el.readyState === el.HAVE_ENOUGH_DATA) {
      canvas.width = el.videoWidth
      canvas.height = el.videoHeight
      ctx.drawImage(el, 0, 0, canvas.width, canvas.height)
      const image = ctx.getImageData(0, 0, canvas.width, canvas.height)
      const result = jsQR(image.data, image.width, image.height, {
        inversionAttempts: 'dontInvert',
      })
      if (result?.data) {
        void handleCode(result.data)
        return
      }
    }
    frame = requestAnimationFrame(tick)
  }

  frame = requestAnimationFrame(tick)
}

async function handleCode(code: string) {
  stopped = true
  scanning.value = false
  pairing.value = true
  status.value = '已识别二维码，正在配对…'

  try {
    const result = await PairByQRCode(code)
    if (result.ok) {
      status.value = result.message
      message.success(result.message)
      emit('paired')
      stop()
      emit('update:show', false)
    } else {
      error.value = result.message || '配对失败'
      status.value = '配对失败，可重新扫描'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
    status.value = '配对失败，可重新扫描'
  } finally {
    pairing.value = false
  }
}

function stop() {
  stopped = true
  if (frame) {
    cancelAnimationFrame(frame)
    frame = 0
  }
  if (stream) {
    stream.getTracks().forEach((track) => track.stop())
    stream = null
  }
  scanning.value = false
}

function close() {
  stop()
  emit('update:show', false)
}

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      void start()
    } else {
      stop()
    }
  },
)

onBeforeUnmount(stop)
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="扫码配对"
    style="width: 520px"
    :bordered="false"
    @update:show="(value: boolean) => (value ? null : close())"
  >
    <div class="qr">
      <div class="preview">
        <video ref="video" class="video" muted playsinline />
        <div v-if="!scanning" class="overlay">
          <span>{{ pairing ? '正在配对…' : '摄像头未开启' }}</span>
        </div>
      </div>

      <p class="status">{{ status }}</p>
      <p v-if="error" class="error">{{ error }}</p>

      <div class="actions">
        <NButton size="small" @click="close">取消</NButton>
        <NButton size="small" type="primary" :loading="pairing" @click="start">开始扫描</NButton>
      </div>

      <p class="tip">
        手机端：设置 → 开发者选项 → 无线调试 → 使用二维码配对设备，将屏幕上的二维码对准摄像头。
      </p>
    </div>
  </NModal>
</template>

<style scoped>
.qr {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.preview {
  position: relative;
  height: 300px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: #000;
}

.video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 12px;
}

.status {
  margin: 0;
  color: var(--text-secondary);
  font-size: 12px;
}

.error {
  margin: 0;
  color: var(--danger);
  font-size: 12px;
  word-break: break-all;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.tip {
  margin: 0;
  color: var(--text-tertiary);
  font-size: 11px;
  line-height: 1.6;
}
</style>
