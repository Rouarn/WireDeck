import { createRouter, createWebHashHistory } from 'vue-router'
import AppsView from '../views/AppsView.vue'
import DevicesView from '../views/DevicesView.vue'
import FilesView from '../views/FilesView.vue'
import InstallView from '../views/InstallView.vue'
import LogsView from '../views/LogsView.vue'
import SettingsView from '../views/SettingsView.vue'
import ShellView from '../views/ShellView.vue'

// Wails 以 file:// 加载页面，必须使用 hash 路由。
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/devices' },
    { path: '/devices', component: DevicesView, meta: { titleKey: 'nav.devices' } },
    { path: '/install', component: InstallView, meta: { titleKey: 'nav.install' } },
    { path: '/apps', component: AppsView, meta: { titleKey: 'nav.apps' } },
    { path: '/files', component: FilesView, meta: { titleKey: 'nav.files' } },
    { path: '/logs', component: LogsView, meta: { titleKey: 'nav.logs' } },
    { path: '/shell', component: ShellView, meta: { titleKey: 'nav.shell' } },
    { path: '/settings', component: SettingsView, meta: { titleKey: 'nav.settings' } },
  ],
})

export default router
