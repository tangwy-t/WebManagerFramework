import type { PluginManifest } from '@/types/plugin'

const plugin: PluginManifest = {
  name: 'system-user',
  version: '1.0.0',
  routes: [
    {
      path: '/system/user',
      name: 'SystemUser',
      component: () => import('./views/index.vue'),
      meta: {
        title: '用户管理',
        icon: 'ri:user-3-line',
        authMark: 'system:user:list'
      }
    }
  ]
}

export default plugin
