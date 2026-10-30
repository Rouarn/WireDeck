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
      message.warning('请选择一个文件')
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
    <div class="picker">
      <div class="bar">
        <NButton size="small" quaternary :title="t('picker.up')" @click="goUp">
          <template #icon><NIcon><ChevronUpOutline /></NIcon></template>
        </NButton>
        <NInput v-model:value="currentPath" size="small" @keyup.enter="load" />
        <NButton size="small" @click="load">{{ t('picker.go') }}</NButton>
      </div>

      <div class="quick">
        <button v-for="path in quickPaths" :key="path" class="quick-chip" @click="jump(path)">
          {{ path }}
        </button>
      </div>

      <div class="files">
        <NSpin :show="loading">
          <div
            v-for="item in files"
            :key="item.path"
            class="row"
            :class="{ dir: item.isDir, selected: selectedPaths.includes(item.path) }"
            @click="onItem(item)"
          >
            <NIcon :size="16" :component="item.isDir ? FolderOutline : DocumentOutline" />
            <span class="name">{{ item.name }}</span>
            <span class="size">{{ item.isDir ? t('picker.dir') : formatSize(item.size) }}</span>
          </div>
          <div v-if="!files.length && !loading" class="empty">{{ t('picker.empty') }}</div>
        </NSpin>
      </div>

      <div class="footer">
        <span class="hint">
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

<style scoped>
.picker {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.bar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.quick {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.quick-chip {
  padding: 3px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  cursor: pointer;
}

.quick-chip:hover {
  border-color: var(--primary-border);
  color: var(--primary-ink);
}

.files {
  height: 320px;
  overflow-y: auto;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-inset);
}

.row {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 8px;
  border-radius: 6px;
  color: var(--text);
  font-size: 12px;
  cursor: pointer;
}

.row:hover {
  background: var(--surface);
}

.row.dir {
  font-weight: 500;
}

.row.selected {
  background: var(--primary-soft);
  color: var(--primary-ink);
}

.row .name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row .size {
  color: var(--text-tertiary);
  font-size: 10px;
}

.empty {
  padding: 40px 0;
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}

.footer {
  display: flex;
  align-items: center;
  gap: 8px;
}

.footer .hint {
  flex: 1;
  color: var(--text-tertiary);
  font-size: 11px;
}
</style>
