<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { GetHostIdentity } from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'
import { useThemeStore } from '../stores/theme'
import { useDevicesStore } from '../stores/devices'
import type { ThemeMode } from '../stores/theme'

const theme = useThemeStore()
const devices = useDevicesStore()
const identity = ref<model.HostIdentity | null>(null)

function onChange(value: ThemeMode) {
  theme.setMode(value)
}

onMounted(async () => {
  try {
    identity.value = await GetHostIdentity()
  } catch {
    identity.value = null
  }
})
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

    <section class="panel">
      <div class="panel-head">
        <h3>配对身份</h3>
        <span class="hint">手机端显示</span>
      </div>
      <div class="panel-body">
        <p class="desc">
          手机「无线调试 → 已配对设备」中显示为
          <code>{{ identity?.hostname || '—' }}@{{ identity?.lanIP || '—' }}</code>。
          @ 前是电脑主机名，@ 后是电脑局域网 IP，不是手机信息。
        </p>
        <p class="desc">该标签由 adb 在配对时上报，配对后不会自动更新。如需显示为 WireDeck，可临时修改主机名：</p>
        <pre class="cmd">sudo scutil --set HostName WireDeck   # 修改
sudo scutil --set HostName ""         # 恢复</pre>
        <p class="desc muted">修改后需在手机上移除旧配对记录并重新配对，才会更新显示。</p>
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

.desc {
  margin: 0 0 10px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.7;
}

.desc code {
  padding: 1px 6px;
  border-radius: 5px;
  background: var(--surface-inset);
  color: var(--text);
}

.desc.muted {
  color: var(--text-tertiary);
  font-size: 11px;
}

.cmd {
  margin: 0 0 10px;
  padding: 10px 12px;
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-inset);
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.6;
}
</style>
