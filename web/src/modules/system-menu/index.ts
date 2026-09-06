import type { PluginManifest } from '@/types/plugin'

const plugin: PluginManifest = {
  name: 'system-menu',
  version: '1.0.0',
  routes: [
    {
      path: '/system/menu',
      name: 'SystemMenu',
      component: () => import('./views/index.vue'),
      meta: {
        title: '菜单管理',
        icon: 'ri:menu-line',
        authMark: 'system:menu:list'
      }
    }
  ]
}

export default plugin
