<script setup lang="ts">
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { useThemeStore } from '../stores/theme'
import { useDevicesStore } from '../stores/devices'
import type { ThemeMode } from '../stores/theme'

const theme = useThemeStore()
const devices = useDevicesStore()

function onChange(value: ThemeMode) {
  theme.setMode(value)
}
</script>

<template>
  <div class="settings">
    <section class="panel">
      <div class="panel-head">
        <h3>外观</h3>
        <span class="hint">切换后立即生效</span>
      </div>
      <div class="panel-body">
        <NRadioGroup :value="theme.mode" @update:value="onChange">
          <NRadioButton value="system">跟随系统</NRadioButton>
          <NRadioButton value="light">浅色</NRadioButton>
          <NRadioButton value="dark">深色</NRadioButton>
        </NRadioGroup>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>adb 运行环境</h3>
      </div>
      <div class="panel-body">
        <div class="row">
          <span class="label">状态</span>
          <span>{{ devices.adb ? '已就绪' : '未就绪' }}</span>
        </div>
        <div class="row">
          <span class="label">路径</span>
          <code>{{ devices.adb?.binaryPath || '—' }}</code>
        </div>
        <div class="row">
          <span class="label">版本</span>
          <span>{{ devices.adb?.version || '—' }}</span>
        </div>
        <div class="row">
          <span class="label">server 端口</span>
          <span>{{ devices.adb?.serverPort || '—' }}</span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 720px;
}

.panel {
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

.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
}

.row:last-child {
  border-bottom: none;
}

.row .label {
  width: 96px;
  color: var(--text-secondary);
}

.row code {
  color: var(--text);
}
</style>
