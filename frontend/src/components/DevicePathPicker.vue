<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NIcon, NInput, NModal, NSpin, useMessage } from 'naive-ui'
import { ChevronUpOutline, DocumentOutline, FolderOutline } from '@vicons/ionicons5'
import { ListDeviceFiles } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { t } from '../i18n'

const props = defineProps<{
  show: boolean
  serial: string
  mode: 'file' | 'dir'
  multi?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'select', path: string): void
  (e: 'select-many', paths: string[]): void
}>()

const message = useMessage()

const currentPath = ref('/sdcard/Download')
const files = ref<model.DeviceFile[]>([])
const loading = ref(false)
const selectedPaths = ref<string[]>([])

const quickPaths = ['/sdcard', '/sdcard/Download', '/sdcard/DCIM', '/sdcard/Pictures', '/data/local/tmp']

const canConfirm = computed(() => props.mode === 'dir' || selectedPaths.value.length > 0)

function formatSize(bytes: number) {
  if (bytes < 1024) {
    return `${bytes} B`
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

async function load() {
  if (!props.serial) {
    return
  }
  loading.value = true
  try {
    files.value = await ListDeviceFiles(props.serial, currentPath.value)
    selectedPaths.value = []
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
    files.value = []
  } finally {
    loading.value = false
  }
}

function onItem(item: model.DeviceFile) {
  if (item.isDir) {
    currentPath.value = item.path
    void load()
    return
  }
  if (props.multi) {
    const index = selectedPaths.value.indexOf(item.path)
    if (index >= 0) {
      selectedPaths.value.splice(index, 1)
    } else {
      selectedPaths.value.push(item.path)
    }
    return
  }
  selectedPaths.value = [item.path]
}

function goUp() {
  if (currentPath.value === '/') {
    return
  }
  currentPath.value = currentPath.value.replace(/\/[^/]+$/, '') || '/'
  void load()
}

function jump(path: string) {
  currentPath.value = path
  void load()
}

function confirm() {
  if (props.mode === 'dir') {
    emit('select', currentPath.value)
  } else {
    if (!selectedPaths.value.length) {
      message.warning(t('picker.selectFileWarning'))
      return
    }
    if (props.multi) {
      emit('select-many', [...selectedPaths.value])
    } else {
      emit('select', selectedPaths.value[0])
    }
  }
  emit('update:show', false)
}

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      void load()
    }
  },
)
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('picker.title')"
    style="width: 680px"
    :bordered="false"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <div class="flex flex-col gap-2.5">
      <div class="flex items-center gap-2">
        <NButton size="small" quaternary :title="t('picker.up')" @click="goUp">
          <template #icon><NIcon><ChevronUpOutline /></NIcon></template>
        </NButton>
        <NInput v-model:value="currentPath" size="small" @keyup.enter="load" />
        <NButton size="small" @click="load">{{ t('picker.go') }}</NButton>
      </div>

      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="path in quickPaths"
          :key="path"
          class="cursor-pointer rounded-full border border-line bg-surface px-2 py-[3px] font-mono text-[10px] text-fg2 hover:border-brand-line hover:text-brand-ink"
          @click="jump(path)"
        >
          {{ path }}
        </button>
      </div>

      <div class="h-[320px] overflow-y-auto rounded-sm border border-line bg-inset p-1">
        <NSpin :show="loading">
          <div
            v-for="item in files"
            :key="item.path"
            class="flex h-8 cursor-pointer items-center gap-2 rounded-[6px] px-2 text-xs text-fg hover:bg-surface"
            :class="[
              item.isDir ? 'font-500' : '',
              selectedPaths.includes(item.path) ? 'bg-brand-soft text-brand-ink' : '',
            ]"
            @click="onItem(item)"
          >
            <NIcon :size="16" :component="item.isDir ? FolderOutline : DocumentOutline" />
            <span class="flex-1 truncate">{{ item.name }}</span>
            <span class="text-[10px] text-fg3">{{ item.isDir ? t('picker.dir') : formatSize(item.size) }}</span>
          </div>
          <div v-if="!files.length && !loading" class="py-10 text-center text-xs text-fg3">
            {{ t('picker.empty') }}
          </div>
        </NSpin>
      </div>

      <div class="flex items-center gap-2">
        <span class="flex-1 text-[11px] text-fg3">
          {{
            mode === 'dir'
              ? t('picker.selectDirHint')
              : multi
                ? t('picker.selectedCount', { n: selectedPaths.length })
                : t('picker.selectFileHint')
          }}
        </span>
        <NButton size="small" @click="emit('update:show', false)">{{ t('common.cancel') }}</NButton>
        <NButton size="small" type="primary" :disabled="!canConfirm" @click="confirm">
          {{ mode === 'dir' ? t('picker.confirmDir') : multi ? t('picker.confirmSelected') : t('picker.confirmFile') }}
        </NButton>
      </div>
    </div>
  </NModal>
</template>
