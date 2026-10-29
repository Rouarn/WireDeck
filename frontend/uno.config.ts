import { defineConfig, presetWind3 } from 'unocss'

export default defineConfig({
  presets: [presetWind3()],
  theme: {
    // 与 design/style.css 的设计令牌保持一致，避免出现第二套色板。
    colors: {
      surface: '#ffffff',
      canvas: '#f6f7f9',
      line: '#e8eaed',
      ink: '#111827',
      'ink-2': '#6b7280',
      brand: '#2563eb',
      success: '#067647',
      warning: '#b54708',
      danger: '#b42318',
    },
  },
  shortcuts: {
    // 常用组合，M1 逐步替换手写样式。
    'wd-card': 'bg-surface border border-line rounded-2.5 p-4 shadow-sm',
    'wd-row': 'flex items-center gap-3',
    'wd-btn': 'h-8 px-3 rounded-2 border border-line bg-surface text-xs cursor-pointer',
  },
})
