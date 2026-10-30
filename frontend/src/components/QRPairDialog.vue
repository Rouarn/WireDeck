<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { NButton, NModal, NSpin, useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import { CreatePairingQR, WaitQRPairing } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { t } from '../i18n'

const props = defineProps<{ show: boolean }>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'paired'): void
}>()

const message = useMessage()

const qrData = ref('')
const status = ref(t('qr.generating'))
const waiting = ref(false)
const error = ref('')
const sessionName = ref('')
const sessionPassword = ref('')

let session: model.QRPairingSession | null = null
let runToken = 0

async function start() {
  const token = ++runToken
  error.value = ''
  waiting.value = true
  status.value = t('qr.generating')

  try {
    session = await CreatePairingQR()
    sessionName.value = session.name
    sessionPassword.value = session.password
    qrData.value = await QRCode.toDataURL(session.payload, {
      width: 280,
      margin: 1,
      errorCorrectionLevel: 'M',
    })
    status.value = t('qr.scanHint')

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
      error.value = result.message || t('qr.notCompleted')
      status.value = t('qr.retryHint')
    }
  } catch (e) {
    if (token === runToken) {
      error.value = e instanceof Error ? e.message : String(e)
      status.value = t('qr.failed')
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
    :title="t('qr.title')"
    style="width: 460px"
    :bordered="false"
    @update:show="(value: boolean) => (value ? null : close())"
  >
    <div class="flex flex-col items-center gap-3">
      <div class="flex h-[300px] w-[300px] items-center justify-center rounded border border-line bg-white">
        <img v-if="qrData" :src="qrData" class="h-[280px] w-[280px]" :alt="t('qr.alt')" />
        <NSpin v-else size="small" />
      </div>

      <p class="m-0 text-center text-xs text-fg2">{{ status }}</p>
      <p v-if="sessionName" class="m-0 text-center font-mono text-[10px] text-fg3">
        {{ t('qr.serviceMeta', { name: sessionName, code: sessionPassword }) }}
      </p>
      <p v-if="error" class="m-0 break-all text-center text-xs text-err">{{ error }}</p>

      <div class="flex gap-2">
        <NButton size="small" @click="close">{{ t('common.cancel') }}</NButton>
        <NButton size="small" type="primary" @click="refresh">{{ t('qr.refresh') }}</NButton>
      </div>

      <p class="m-0 text-center text-[11px] leading-relaxed text-fg3">
        {{ t('qr.hint') }}
      </p>
    </div>
  </NModal>
</template>


