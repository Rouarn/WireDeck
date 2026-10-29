import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export type ThemeMode = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'wiredeck.theme'

export const useThemeStore = defineStore('theme', () => {
  const stored = localStorage.getItem(STORAGE_KEY)
  const mode = ref<ThemeMode>(stored === 'light' || stored === 'dark' ? stored : 'system')
  const systemDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)

  const resolved = computed<'light' | 'dark'>(() =>
    mode.value === 'system' ? (systemDark.value ? 'dark' : 'light') : mode.value,
  )
  const isDark = computed(() => resolved.value === 'dark')

  function apply() {
    document.documentElement.dataset.theme = isDark.value ? 'dark' : 'light'
  }

  function setMode(next: ThemeMode) {
    mode.value = next
    localStorage.setItem(STORAGE_KEY, next)
    apply()
  }

  let initialized = false
  function init() {
    if (initialized) {
      return
    }
    initialized = true

    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', (event) => {
      systemDark.value = event.matches
      apply()
    })
    apply()
  }

  return { mode, isDark, setMode, init }
})
