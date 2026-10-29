<script setup lang="ts">
import { computed } from 'vue'
import { NConfigProvider, NDialogProvider, NMessageProvider, darkTheme } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'
import AppLayout from './layouts/AppLayout.vue'
import { useThemeStore } from './stores/theme'

const theme = useThemeStore()
theme.init()

const naiveTheme = computed(() => (theme.isDark ? darkTheme : null))

const overrides = computed<GlobalThemeOverrides>(() => ({
  common: {
    primaryColor: theme.isDark ? '#3B82F6' : '#2563EB',
    primaryColorHover: theme.isDark ? '#60A5FA' : '#1D4ED8',
    primaryColorPressed: theme.isDark ? '#2563EB' : '#1E40AF',
    borderRadius: '8px',
    fontSize: '13px',
    fontFamily:
      "-apple-system, BlinkMacSystemFont, 'PingFang SC', 'Helvetica Neue', 'Microsoft YaHei', Arial, sans-serif",
  },
}))
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="overrides">
    <NMessageProvider>
      <NDialogProvider>
        <AppLayout />
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
