import { ref } from 'vue'

export type Lang = 'zh' | 'en'

const STORAGE_KEY = 'wiredeck.lang'

const messages = {
  zh: {
    'nav.devices': '设备',
    'nav.install': '安装台',
    'nav.apps': '应用',
    'nav.files': '文件',
    'nav.logs': '日志',
    'nav.shell': 'Shell',
    'nav.settings': '设置',

    'common.refresh': '刷新',
    'common.select': '选择',
    'common.cancel': '取消',
    'common.clear': '清空',
    'common.unknown': '未选择设备',
    'common.adbNotReady': 'adb 未就绪',

    'settings.title': '设置',
    'settings.appearance': '外观',
    'settings.appearanceHint': '切换后立即生效',
    'settings.theme.system': '跟随系统',
    'settings.theme.light': '浅色',
    'settings.theme.dark': '深色',
    'settings.language': '语言 / Language',
    'settings.languageHint': '切换界面语言，立即生效',
    'settings.adb': 'adb 运行环境',
    'settings.adbStatus': '状态',
    'settings.adbPath': '路径',
    'settings.adbVersion': '版本',
    'settings.adbPort': 'server 端口',
    'settings.ready': '已就绪',
    'settings.notReady': '未就绪',
    'settings.pairing': '配对身份',
    'settings.pairingHint': '手机端显示',
    'settings.cache': '应用名称缓存',
    'settings.cacheHint': '解析后的应用中文名',
    'settings.cacheClear': '清理缓存',
    'settings.update': '版本更新',
    'settings.updateHint': '当前 v{version}',
    'settings.checkUpdate': '检查更新',
    'settings.openDownload': '打开下载页',
    'settings.upToDate': '已是最新版本',
    'settings.newVersion': '发现新版本 v{version}',
    'settings.notChecked': '未检查',
  },
  en: {
    'nav.devices': 'Devices',
    'nav.install': 'Install',
    'nav.apps': 'Apps',
    'nav.files': 'Files',
    'nav.logs': 'Logs',
    'nav.shell': 'Shell',
    'nav.settings': 'Settings',

    'common.refresh': 'Refresh',
    'common.select': 'Select',
    'common.cancel': 'Cancel',
    'common.clear': 'Clear',
    'common.unknown': 'No device selected',
    'common.adbNotReady': 'adb not ready',

    'settings.title': 'Settings',
    'settings.appearance': 'Appearance',
    'settings.appearanceHint': 'Takes effect immediately',
    'settings.theme.system': 'System',
    'settings.theme.light': 'Light',
    'settings.theme.dark': 'Dark',
    'settings.language': 'Language / 语言',
    'settings.languageHint': 'Switch interface language immediately',
    'settings.adb': 'adb environment',
    'settings.adbStatus': 'Status',
    'settings.adbPath': 'Path',
    'settings.adbVersion': 'Version',
    'settings.adbPort': 'Server port',
    'settings.ready': 'Ready',
    'settings.notReady': 'Not ready',
    'settings.pairing': 'Pairing identity',
    'settings.pairingHint': 'Shown on the phone',
    'settings.cache': 'App name cache',
    'settings.cacheHint': 'Resolved app display names',
    'settings.cacheClear': 'Clear cache',
    'settings.update': 'Updates',
    'settings.updateHint': 'Current v{version}',
    'settings.checkUpdate': 'Check for updates',
    'settings.openDownload': 'Open download page',
    'settings.upToDate': 'You are up to date',
    'settings.newVersion': 'New version v{version}',
    'settings.notChecked': 'Not checked',
  },
} as const

const lang = ref<Lang>((localStorage.getItem(STORAGE_KEY) as Lang) || 'zh')

export function t(key: string, params?: Record<string, string | number>): string {
  const table = messages[lang.value] ?? messages.zh
  let text: string = (table as Record<string, string>)[key] ?? (messages.zh as Record<string, string>)[key] ?? key
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      text = text.replaceAll(`{${k}}`, String(v))
    }
  }
  return text
}

export function useLang() {
  function setLang(value: Lang) {
    lang.value = value
    localStorage.setItem(STORAGE_KEY, value)
  }
  return { lang, setLang, t }
}
