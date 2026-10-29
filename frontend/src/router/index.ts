import { createRouter, createWebHashHistory } from 'vue-router'
import AppsView from '../views/AppsView.vue'
import DevicesView from '../views/DevicesView.vue'
import FilesView from '../views/FilesView.vue'
import InstallView from '../views/InstallView.vue'
import LogsView from '../views/LogsView.vue'
import PlaceholderView from '../views/PlaceholderView.vue'
import SettingsView from '../views/SettingsView.vue'

// Wails 以 file:// 加载页面，必须使用 hash 路由。
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/devices' },
    { path: '/devices', component: DevicesView, meta: { title: '设备' } },
    { path: '/install', component: InstallView, meta: { title: '安装台' } },
    { path: '/apps', component: AppsView, meta: { title: '应用' } },
    { path: '/files', component: FilesView, meta: { title: '文件' } },
    { path: '/logs', component: LogsView, meta: { title: '日志' } },
    { path: '/settings', component: SettingsView, meta: { title: '设置' } },
  ],
})

export default router
