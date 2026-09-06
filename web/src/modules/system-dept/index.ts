import type { PluginManifest } from '@/types/plugin'

const plugin: PluginManifest = {
  name: 'system-dept',
  version: '1.0.0',
  routes: [
    {
      path: '/system/dept',
      name: 'SystemDept',
      component: () => import('./views/index.vue'),
      meta: {
        title: '部门管理',
        icon: 'ri:organization-chart',
        authMark: 'system:dept:list'
      }
    }
  ]
}

export default plugin
