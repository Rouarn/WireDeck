import { createRouter, createWebHashHistory } from 'vue-router'
import DevicesView from '../views/DevicesView.vue'
import InstallView from '../views/InstallView.vue'
import PlaceholderView from '../views/PlaceholderView.vue'
import SettingsView from '../views/SettingsView.vue'

// Wails 以 file:// 加载页面，必须使用 hash 路由。
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/devices' },
    { path: '/devices', component: DevicesView, meta: { title: '设备' } },
    { path: '/install', component: InstallView, meta: { title: '安装台' } },
    {
      path: '/apps',
      component: PlaceholderView,
      meta: { title: '应用', description: 'M3 将在此实现应用列表、卸载与导出。' },
    },
    {
      path: '/files',
      component: PlaceholderView,
      meta: { title: '文件', description: 'M3 将在此实现文件推送、拉取与截图。' },
    },
    {
      path: '/logs',
      component: PlaceholderView,
      meta: { title: '日志', description: 'M4 将在此实现 logcat 实时查看与过滤。' },
    },
    { path: '/settings', component: SettingsView, meta: { title: '设置' } },
  ],
})

export default router
