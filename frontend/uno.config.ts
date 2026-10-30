import { defineConfig, presetWind3 } from 'unocss'

export default defineConfig({
  presets: [presetWind3()],
  theme: {
    // 颜色全部映射到 theme.css 的 CSS 变量，保持明暗两套主题一致。
    colors: {
      bg: 'var(--bg)',
      surface: 'var(--surface)',
      muted: 'var(--surface-muted)',
      inset: 'var(--surface-inset)',
      line: 'var(--border)',
      'line-strong': 'var(--border-strong)',
      fg: 'var(--text)',
      fg2: 'var(--text-secondary)',
      fg3: 'var(--text-tertiary)',
      brand: 'var(--primary)',
      'brand-hover': 'var(--primary-hover)',
      'brand-soft': 'var(--primary-soft)',
      'brand-line': 'var(--primary-border)',
      'brand-ink': 'var(--primary-ink)',
      ok: 'var(--success)',
      warn: 'var(--warning)',
      err: 'var(--danger)',
      'err-soft': 'var(--danger-soft)',
    },
    radius: {
      DEFAULT: '10px',
      sm: '8px',
    },
    boxShadow: {
      sm: 'var(--shadow-sm)',
      md: 'var(--shadow-md)',
    },
  },
  shortcuts: {
    panel: 'border border-line rounded bg-surface shadow-sm',
    'panel-head': 'flex items-center gap-2 px-4 py-3 border-b border-line',
    'panel-title': 'm-0 text-[13px] font-600',
    'panel-body': 'p-4',
    'field-label': 'text-fg3 text-[11px]',
    'toolbar-btn':
      'h-8 px-3 rounded-sm border border-line-strong bg-surface text-fg text-xs cursor-pointer hover:bg-muted',
  },
})
