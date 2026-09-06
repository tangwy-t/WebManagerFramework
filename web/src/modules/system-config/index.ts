import type { PluginManifest } from '@/types/plugin'

const plugin: PluginManifest = {
  name: 'system-config',
  version: '1.0.0',
  routes: [
    {
      path: '/system/config',
      name: 'SystemConfig',
      component: () => import('./views/index.vue'),
      meta: {
        title: '参数配置',
        icon: 'ri:settings-3-line',
        authMark: 'system:config:list'
      }
    }
  ]
}

export default plugin
